package requestid_test

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/pkg/requestid"
)

// Example_other show how to set the interceptors of requestid on a gRPC server
func Example_other() {
	server := grpc.NewServer(
		// Can be used directly
		grpc.UnaryInterceptor(requestid.UnaryServerInterceptor),
		// Or chained with other interceptors
		grpc.ChainStreamInterceptor(
			requestid.StreamServerInterceptor,
			// other interceptors...
		),
	)
	foobar.RegisterDummyServiceServer(server, &foobar.UnimplementedDummyServiceServer{})
}

// ExampleGetFromContext shows how to get the request identifier in a gRPC method handler
func ExampleGetFromContext() {
	requestId := requestid.GetFromContext(ctx)
	fmt.Println(requestId)
}

// ExampleAppendToOutgoingContext shows how to send a request correlation identifier to a client gRPC call
func ExampleAppendToOutgoingContext() {
	conn, err := grpc.DialContext(ctx, "127.0.0.1:1234")
	if err != nil {
		panic(err)
	}
	client := foobar.NewDummyServiceClient(conn)

	// Add a request id to call
	ctx = requestid.AppendToOutgoingContext(ctx, "i'm an unique id")
	var header metadata.MD
	_, err = client.Foo(ctx, &foobar.Empty{}, grpc.Header(&header))
	// as request correlation identifier is sent in response header,
	// requestid.GetFromMeta(header) => "i'm an unique id"
}

var ctx context.Context
