package ports

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/SinghAman21/openport/internal/model"
)

// DefaultSuggestNear is where suggestions start when the caller has no
// preference — the conventional dev-server neighbourhood.
const DefaultSuggestNear = 3000

// maxSuggestScan bounds how far past the hint we look before giving up, so a
// machine with a pathological number of listeners cannot hang the UI.
const maxSuggestScan = 2000

// SuggestFree returns up to count ports at or above near that are free. A port
// qualifies only if nothing is listening on it, it is not reserved by an alias,
// it is not protected, and it can actually be bound right now.
func (s *service) SuggestFree(ctx context.Context, near, count int) ([]int, error) {
	if count <= 0 {
		count = 1
	}
	if near <= 0 {
		near = DefaultSuggestNear
	}
	if err := model.ValidatePort(near); err != nil {
		return nil, err
	}

	socks, err := s.prov.ListListening(ctx)
	if err != nil {
		return nil, err
	}
	occupied := make(map[int]bool, len(socks))
	for _, sock := range socks {
		occupied[sock.Port] = true
	}

	e := s.loadEnrichment()
	// Aliases double as reservations, so a port someone has claimed for a
	// service that is not running yet still will not be suggested.
	for port := range e.aliases {
		occupied[port] = true
	}

	out := make([]int, 0, count)
	for port := near; port <= 65535 && port < near+maxSuggestScan && len(out) < count; port++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if occupied[port] {
			continue
		}
		if protected, _ := e.protected.IsProtected(port, ""); protected {
			continue
		}
		if !s.canBind(port) {
			continue
		}
		out = append(out, port)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no free port found within %d of %d", maxSuggestScan, near)
	}

	s.record(model.HistoryEntry{
		At:      s.now(),
		Kind:    model.ActionSuggest,
		Summary: fmt.Sprintf("suggested %s near %d", joinInts(out), near),
		Port:    out[0],
		Success: true,
	})
	return out, nil
}

// canBindLocal reports whether a TCP port can be bound right now. This catches
// sockets the listing missed — another user's process, or a port held in
// TIME_WAIT. Injected into the service so tests never touch the network.
func canBindLocal(port int) bool {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func joinInts(xs []int) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += strconv.Itoa(x)
	}
	return out
}

// suggestTimeout bounds a single suggestion sweep.
const suggestTimeout = 5 * time.Second
