// Package remoteaddr gets the gRPC client's remote address.
package remoteaddr

import (
	"context"
	"errors"
	"fmt"
	"net"

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
