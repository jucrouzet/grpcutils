package utils

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
)

func testDialer(impl foobar.DummyServiceServer, opts ...grpc.ServerOption) func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer(opts...)

	foobar.RegisterDummyServiceServer(server, impl)

	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Errorf("Failed launching DummyService server : %w", err))
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestCallFoo(
	t *testing.T,
	impl foobar.DummyServiceServer,
	clientOpts []grpc.DialOption,
	serverOpts []grpc.ServerOption,
	clientContext ...context.Context,
) (*foobar.Empty, metadata.MD, metadata.MD, error) {
	ctx := context.TODO()
	if len(clientContext) > 0 {
		ctx = clientContext[0]
	}
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(testDialer(impl, serverOpts...)),
	}
	opts = append(opts, clientOpts...)
	conn, err := grpc.DialContext(ctx, "", opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("Failed creating DummyService : %w", err))
	}
	client := foobar.NewDummyServiceClient(conn)
	var header, trailer metadata.MD
	v, err := client.Foo(ctx, &foobar.Empty{}, grpc.Header(&header), grpc.Trailer(&trailer))
	return v, header, trailer, err
}

func TestCallFooS(
	t *testing.T,
	impl foobar.DummyServiceServer,
	clientOpts []grpc.DialOption,
	serverOpts []grpc.ServerOption,
	clientContext ...context.Context,
) (metadata.MD, metadata.MD) {
	ctx := context.TODO()
	if len(clientContext) > 0 {
		ctx = clientContext[0]
	}
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(testDialer(impl, serverOpts...)),
	}
	opts = append(opts, clientOpts...)
	conn, err := grpc.DialContext(ctx, "", opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("Failed creating DummyService : %w", err))
	}
	client := foobar.NewDummyServiceClient(conn)
	var header, trailer metadata.MD
	s, err := client.FooS(ctx, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		t.Fatal(fmt.Errorf("Failed creating DummyService FooS server : %w", err))
	}
	for i := 0; i <= 5; i++ {
		err = s.Send(&foobar.Empty{})
		if err != nil {
			t.Fatal(fmt.Errorf("Failed sending message on DummyService FooS server : %w", err))
		}
	}
	err = s.CloseSend()
	if err != nil {
		t.Fatal(fmt.Errorf("Failed CloseSend() on DummyService FooS server : %w", err))
	}
	for {
		_, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return header, trailer
}
