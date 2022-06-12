package requestid

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
)

type dummyRequestID struct {
	foobar.UnimplementedDummyServiceServer
	t           *testing.T
	expectingID string
}

func (d *dummyRequestID) Foo(ctx context.Context, in *foobar.Empty) (*foobar.Empty, error) {
	id := GetFromContext(ctx)
	assert.NotEmpty(d.t, id, "UnaryServerInterceptor should have set a request correlation identifier metadata")
	if d.expectingID != "" {
		assert.Equal(d.t, d.expectingID, id, "UnaryServerInterceptor should have set the request correlation identifier metadata sent in request")
	}
	return &foobar.Empty{}, nil
}

func (d *dummyRequestID) FooS(s foobar.DummyService_FooSServer) error {
	for {
		_, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			d.t.Fatal(err)
		}
		id := GetFromContext(s.Context())
		assert.NotEmpty(d.t, id, "StreamServerInterceptorRequestID should have set a request correlation identifier metadata")
		if d.expectingID != "" {
			assert.Equal(d.t, d.expectingID, id, "StreamServerInterceptorRequestID should have set the request correlation identifier metadata sent in request")
		}
	}
	return nil
}

func TestGetFromContext(t *testing.T) {
	ctx := context.TODO()
	id := GetFromContext(ctx)
	assert.Equal(t, "", id, "GetFromContext() should returns an empty string when the request correlation identifier when metadata is not set in IncomingContext")

	ctx = context.TODO()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(MetadataName, "a"))
	id = GetFromContext(ctx)
	assert.Equal(t, "a", id, "GetFromContext() should returns the request correlation identifier when metadata is set in IncomingContext")

	ctx = context.TODO()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(MetadataName, "1", MetadataName, "2", MetadataName, "3"))
	id = GetFromContext(ctx)
	assert.Equal(t, "1", id, "GetFromContext() should returns the first request correlation identifier when metadata is set multiple times")
}

func TestUnaryServerInterceptor(t *testing.T) {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(UnaryServerInterceptor),
	}
	_, header, _, _ := utils.TestCallFoo(t, &dummyRequestID{t: t}, nil, opts)
	assert.NotEmpty(t, GetFromMeta(header), "UnaryServerInterceptor should have set a request correlation identifier metadata in response header")
	_, header, _, _ = utils.TestCallFoo(t, &dummyRequestID{t: t, expectingID: "coucou"}, nil, opts, AppendToOutgoingContext(context.TODO(), "coucou"))
	assert.Equal(t, "coucou", GetFromMeta(header), "UnaryServerInterceptor should have set the valid request correlation identifier metadata in response header")
}

func TestStreamServerInterceptorRequestID(t *testing.T) {
	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(StreamServerInterceptor),
	}
	header, _ := utils.TestCallFooS(t, &dummyRequestID{t: t}, nil, opts)
	assert.NotEmpty(t, GetFromMeta(header), "StreamServerInterceptorRequestID should have set a request correlation identifier metadata in response header")
	header, _ = utils.TestCallFooS(t, &dummyRequestID{t: t, expectingID: "coucou2"}, nil, opts, AppendToOutgoingContext(context.TODO(), "coucou2"))
	assert.Equal(t, "coucou2", GetFromMeta(header), "UnaryServerInterceptor should have set the valid request correlation identifier metadata in response header")
}

func TestAppendToOutgoingContext(t *testing.T) {
	ctx := context.TODO()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(MetadataName, "before"))
	ctx = AppendToOutgoingContext(context.TODO(), "test")
	md, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok, "context returned by AppendToOutgoingContext() should have metadata")
	vals := md.Get(MetadataName)
	assert.Equal(t, 1, len(vals), "context returned by AppendToOutgoingContext() should have set only one request correlation identifier metadata")
	assert.Equal(t, "test", vals[0], "context returned by AppendToOutgoingContext() should have set a request correlation identifier metadata")

	ctx2 := AppendToOutgoingContext(context.TODO())
	md, ok = metadata.FromOutgoingContext(ctx2)
	assert.True(t, ok, "context returned by AppendToOutgoingContext() without a value should have metadata")
	vals = md.Get(MetadataName)
	assert.Equal(t, 1, len(vals), "context returned by AppendToOutgoingContext() without a value should have set a request correlation identifier metadata")
	assert.NotEqual(t, "", vals[0], "context returned by AppendToOutgoingContext() without a value should have set a request correlation identifier metadata")
}
