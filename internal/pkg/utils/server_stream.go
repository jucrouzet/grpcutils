package utils

import (
	"context"

	"google.golang.org/grpc"
)

// ServerStream composes a new grpc stream server with a new context
type ServerStream struct {
	grpc.ServerStream
	Ctx context.Context
}

// Context returns the server's context
func (s *ServerStream) Context() context.Context {
	return s.Ctx
}
