// Package remoteaddr gets the gRPC client's remote address.
package remoteaddr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc/peer"
)

var (
	// ErrNotAvailable is returned when the remote client address is not available
	ErrNotAvailable = errors.New("remote address is not available")
)

// GetFromContext returns the remote address of the client calling the method, for both unary
// and streaming method, in a gRPC call context.
// ErrNotAvailable is returned if remote address is not available.
func GetFromContext(ctx context.Context) (net.Addr, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("%w: no peer infos", ErrNotAvailable)
	}
	if pr.Addr == net.Addr(nil) {
		return nil, fmt.Errorf("%w: nil address", ErrNotAvailable)
	}
	return pr.Addr, nil
}

// GetIPFromContext returns the remote ip of the client calling the method, for both unary
// and streaming method, in a gRPC call context.
// ErrNotAvailable is returned if remote address is not available.
func GetIPFromContext(ctx context.Context) (net.IP, error) {
	addr, err := GetFromContext(ctx)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(addr.String(), ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: could not retreive remote address (invalid address format)", ErrNotAvailable)
	}
	ip := net.ParseIP(strings.Join(parts[:len(parts)-1], ":"))
	if ip == nil {
		return nil, fmt.Errorf("%w: could not retreive remote address (invalid IP)", ErrNotAvailable)
	}
	return ip, nil
}
