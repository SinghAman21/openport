package ports

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/SinghAman21/openport/internal/model"
)

// DefaultWatchInterval is the refresh cadence for watch mode.
const DefaultWatchInterval = 2 * time.Second

// Watch polls the host on an interval and publishes each listing on the
// returned channel. The channel holds at most one pending listing: if the
// consumer is busy, the stale entry is dropped in favour of the fresh one, so
// the poller never blocks and the UI never renders a backlog.
//
// The returned func cancels the watch; the channel is closed once the poller
// has stopped, which is what tells the UI's re-arming command to stand down.
func (s *service) Watch(ctx context.Context, interval time.Duration) (<-chan []Port, func()) {
	if interval <= 0 {
		interval = DefaultWatchInterval
	}
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan []Port, 1)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		emit := func() {
			list, err := s.List(ctx)
			if err != nil {
				if ctx.Err() == nil {
					s.logf(model.LogWarn, "watch refresh failed: %v", err)
				}
				return
			}
			select {
			case ch <- list:
			default:
				// Consumer is behind. Drop the stale listing, keep the fresh
				// one — showing the newest state matters more than showing
				// every intermediate one.
				select {
				case <-ch:
				default:
				}
				select {
				case ch <- list:
				default:
				}
			}
		}

		emit() // publish immediately so the UI does not wait a full interval
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				emit()
			}
		}
	}()

	return ch, cancel
}

// asOpError is errors.As for *net.OpError, kept here so health.go does not
// need to import errors directly.
func asOpError(err error, target **net.OpError) bool { return errors.As(err, target) }
