// Package requestid handles a unique request correlation identifier via metadata, similary to the
// `X-Request-Id` header in HTTP.
package requestid

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
)

const (
	// MetadataName is the name of the metadata that holds an unique identifier for the call.
	MetadataName = "request-id"
)

// UnaryServerInterceptor is a server unary interceptor that ensures that method calls has
// a request correlation identifier metadata, adds it if not, and add a request correlation identifier
// metadata to response's header.
func UnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	id := GetFromContext(ctx)
	if id == "" {
		id = uuid.New().String()
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	md.Set(MetadataName, id)
	ctx = metadata.NewIncomingContext(ctx, md)
	if err := grpc.SetHeader(ctx, metadata.Pairs(MetadataName, id)); err != nil {
		return nil, status.Error(codes.Internal, "failed setting request correlation identifier")
	}
	return handler(ctx, req)
}

// StreamServerInterceptor is a server stream interceptor that ensures that method calls has
// a request correlation identifier metadata, adds it if not, and add a request correlation
// identifier metadata to response's header.
func StreamServerInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	id := GetFromContext(stream.Context())
	if id == "" {
		id = uuid.New().String()
	}
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		md = metadata.MD{}
	}
	md.Set(MetadataName, id)
	ctx := metadata.NewIncomingContext(stream.Context(), md)

	ns := &utils.ServerStream{
		ServerStream: stream,
		Ctx:          ctx,
	}
	if err := stream.SetHeader(metadata.Pairs(MetadataName, id)); err != nil {
		return status.Error(codes.Internal, "failed setting request correlation identifier")
	}
	return handler(srv, ns)
}

// GetFromContext returns the request correlation identifier from a gRPC incoming context
func GetFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return GetFromMeta(md)
}

// GetFromMeta returns the request correlation identifier from a gRPC metadata map
func GetFromMeta(md metadata.MD) string {
	if len(md.Get(MetadataName)) < 1 {
		return ""
	}
	return md.Get(MetadataName)[0]
}

// AppendToOutgoingContext generates an outgoing gRPC context with a correlation identifier.
// If `id` is passed, it will be used, else a random identifier will be generated.
func AppendToOutgoingContext(ctx context.Context, id ...string) context.Context {
	var requestID string
	if len(id) == 0 {
		requestID = uuid.New().String()
	} else {
		requestID = id[0]
	}
	return metadata.AppendToOutgoingContext(ctx, MetadataName, requestID)
}
