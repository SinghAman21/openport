package ports

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/SinghAman21/openport/internal/model"
)

// DefaultHealthTimeout bounds a single probe. Kept short: a health check that
// blocks the UI for seconds is worse than one that reports "down".
const DefaultHealthTimeout = 2 * time.Second

// prober isolates the actual I/O so the service can be tested without opening
// sockets. netProber is the real implementation.
type prober interface {
	TCP(ctx context.Context, host string, port int, timeout time.Duration) error
	HTTP(ctx context.Context, url string, timeout time.Duration) (int, error)
}

type netProber struct{}

func (netProber) TCP(ctx context.Context, host string, port int, timeout time.Duration) error {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	return conn.Close()
}

func (netProber) HTTP(ctx context.Context, url string, timeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	// No redirect following: a 302 is a meaningful answer about the service.
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// Health probes a port over TCP or HTTP and reports the outcome. Probe
// failures are results, not errors: "the service is down" is the answer the
// caller asked for. A non-nil error means the probe could not be attempted.
func (s *service) Health(ctx context.Context, port int, kind HealthKind) (HealthResult, error) {
	if err := model.ValidatePort(port); err != nil {
		return HealthResult{}, err
	}

	start := s.now()
	res := HealthResult{CheckedAt: start}

	switch kind {
	case HealthHTTP:
		url := fmt.Sprintf("http://127.0.0.1:%d/", port)
		code, err := s.probe.HTTP(ctx, url, DefaultHealthTimeout)
		res.LatencyMS = int(s.now().Sub(start) / time.Millisecond)
		res.Code = code
		if err != nil {
			res.Status = shortErr(err)
			return res, nil
		}
		res.OK = code >= 200 && code < 400
		res.Status = http.StatusText(code)
		if res.Status == "" {
			res.Status = fmt.Sprintf("HTTP %d", code)
		}
	default:
		err := s.probe.TCP(ctx, "127.0.0.1", port, DefaultHealthTimeout)
		res.LatencyMS = int(s.now().Sub(start) / time.Millisecond)
		if err != nil {
			res.Status = shortErr(err)
			return res, nil
		}
		res.OK = true
		res.Status = "open"
	}
	return res, nil
}

// shortErr trims the noisy dial prefixes off a probe failure so the badge
// tooltip stays readable.
func shortErr(err error) string {
	var opErr *net.OpError
	if ok := asOpError(err, &opErr); ok && opErr.Err != nil {
		return opErr.Err.Error()
	}
	return err.Error()
}
