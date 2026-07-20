//go:build windows

package platform

import "context"

// windowsProvider is a compiling stub. Wiring Windows up means implementing
// these three methods (GetExtendedTcpTable / GetExtendedUdpTable via
// iphlpapi.dll) — nothing else in the codebase has to change.
type windowsProvider struct{ log LogFunc }

func newProvider(log LogFunc) (Provider, error) {
	return nil, ErrUnsupported
}

func (p *windowsProvider) ListListening(ctx context.Context) ([]Socket, error) {
	return nil, ErrUnsupported
}

func (p *windowsProvider) ProcessOnPort(ctx context.Context, port int) (Socket, error) {
	return Socket{}, ErrUnsupported
}

func (p *windowsProvider) Kill(ctx context.Context, pid int, sig Signal) error {
	return ErrUnsupported
}
