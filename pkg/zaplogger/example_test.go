package zaplogger_test

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/pkg/zaplogger"
)

// ExampleNew creates a new zaplogger
func ExampleNew() {
	l, err := zaplogger.New(
		zaplogger.WithServerName("foobar service"),
		zaplogger.WithFields(
			zaplogger.FieldMethod,
			zaplogger.FieldRemoteAddr,
			zaplogger.FieldServerName,
		),
	)
	if err != nil {
		panic(err)
	}
	l.GetLogger().Debug("hello world")
}

// ExampleWithLogger creates a new zaplogger specifying the zap logger
func ExampleWithLogger() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	a, err := zaplogger.New(
		zaplogger.WithLogger(logger),
		zaplogger.WithServerName("foobar service"),
	)
	if err != nil {
		panic(err)
	}
	a.GetLogger().Debug("hello world")
}

// Example_otherInterceptors shows how to use the interceptors of zaplogger
func Example_otherInterceptors() {
	l, err := zaplogger.New()
	if err != nil {
		panic(err)
	}

	// Interceptor can be used directly
	server := grpc.NewServer(
		grpc.UnaryInterceptor(l.UnaryInterceptor()),
		grpc.StreamInterceptor(l.StreamInterceptor()),
	)
	// Or chained with other interceptors
	server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// UnaryInterceptor
			l.UnaryInterceptor(),
			// other interceptor
		),
		grpc.ChainStreamInterceptor(
			// StreamInterceptor
			l.StreamInterceptor(),
			// other interceptor
		),
	)
	foobar.RegisterDummyServiceServer(server, &foobar.UnimplementedDummyServiceServer{})
}

func ExampleGetFromContext_unary() {
	// in the body of a service method like :
	// func MyServiceUnaryMethod(ctx context.Context, param service.Type) (service.Type, error) {
	logger, err := zaplogger.GetFromContext(ctx)
	if err != nil {
		panic(err)
	}
	logger.With(zap.String("foo", "bar")).Info("important message")
}

func ExampleGetFromContext_stream() {
	// in the body of a service method like :
	// func MyServiceStreamMethod(s service.Service_MethodServer) error {
	logger, err := zaplogger.GetFromContext(s.Context())
	if err != nil {
		panic(err)
	}
	logger.With(zap.String("foo", "bar")).Info("important message")
}

var ctx context.Context
var s foobar.DummyService_FooSServer
