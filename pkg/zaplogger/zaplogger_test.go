package zaplogger

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
)

func TestWithLogger(t *testing.T) {
	l, err := New(WithLogger(nil))
	assert.ErrorIs(t, err, ErrInvalidOptionValue, "using a nil logger should return an ErrInvalidOptionValue")
	assert.Nil(t, l, "using a nil logger should return a nil logger")

	core, recordedLogs := observer.New(zapcore.DebugLevel)
	l, _ = New(WithLogger(zap.New(core)))
	l.GetLogger().Debug("test")
	assert.Equal(t, 1, len(recordedLogs.All()), "WithLogger() should set the logger")
}

type dummyLoggerGetFromContext struct {
	foobar.UnimplementedDummyServiceServer
	t *testing.T
}

func (d *dummyLoggerGetFromContext) Foo(ctx context.Context, in *foobar.Empty) (*foobar.Empty, error) {
	logger, err := GetFromContext(ctx)
	assert.IsType(d.t, &zap.Logger{}, logger, "GetFromContext() should return a *zap.Logger when UnaryInterceptor is in use")
	assert.Nil(d.t, err, "GetFromContext() should not return an error when UnaryInterceptor is in use")
	return &foobar.Empty{}, nil
}

func (d *dummyLoggerGetFromContext) FooS(s foobar.DummyService_FooSServer) error {
	logger, err := GetFromContext(s.Context())
	assert.IsType(d.t, &zap.Logger{}, logger, "GetFromContext() should return a *zap.Logger when UnaryInterceptor is in use")
	assert.Nil(d.t, err, "GetFromContext() should not return an error when UnaryInterceptor is in use")
	for {
		_, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			d.t.Fatal(err)
		}
	}
	return nil
}

func TestWdummyLoggerGetFromContext(t *testing.T) {
	l, _ := New()
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(l.UnaryInterceptor()),
		grpc.StreamInterceptor(l.StreamInterceptor()),
	}
	utils.TestCallFoo(t, &dummyLoggerGetFromContext{t: t}, nil, opts)
	utils.TestCallFooS(t, &dummyLoggerGetFromContext{t: t}, nil, opts)
	logger, err := GetFromContext(context.TODO())
	assert.Nil(t, logger, "GetFromContext() should not return a *zap.Logger when context has no logger")
	assert.ErrorIs(t, err, ErrNoLoggerInContext, "GetFromContext() should return a ErrNoLoggerInContext error when context has no logger")

}

type dummyLoggerWithFields struct {
	foobar.UnimplementedDummyServiceServer
	t    *testing.T
	logs *observer.ObservedLogs
}

func (d *dummyLoggerWithFields) Foo(ctx context.Context, in *foobar.Empty) (*foobar.Empty, error) {
	logger, err := GetFromContext(ctx)
	assert.IsType(d.t, &zap.Logger{}, logger, "GetFromContext() should return a *zap.Logger when UnaryInterceptor is in use")
	assert.Nil(d.t, err, "GetFromContext() should not return an error when UnaryInterceptor is in use")
	logger.Debug("test")
	assert.Equal(d.t, 1, len(d.logs.All()), "there should be a log message")
	var okFields []string
	if len(d.logs.All()[0].Context) > 0 {

		for _, f := range d.logs.All()[0].Context {
			if f.Key == FieldRemoteAddr {
				assert.Equal(d.t, "bufconn", f.String, "remote addr has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "remote addr has a wrong type")
				okFields = append(okFields, FieldRemoteAddr)
			}
			if f.Key == FieldServerType {
				assert.Equal(d.t, "*zaplogger.dummyLoggerWithFields", f.String, "server type has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "server type has a wrong type")
				okFields = append(okFields, FieldServerType)
			}
			if f.Key == FieldMethod {
				assert.Equal(d.t, "/foobar.DummyService/Foo", f.String, "method has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "method has a wrong type")
				okFields = append(okFields, FieldMethod)
			}
		}
	}
	assert.Contains(d.t, okFields, FieldServerType, "expected server type field")
	assert.Contains(d.t, okFields, FieldRemoteAddr, "expected remote addr field")
	return &foobar.Empty{}, nil
}

func (d *dummyLoggerWithFields) FooS(s foobar.DummyService_FooSServer) error {
	logger, err := GetFromContext(s.Context())
	assert.IsType(d.t, &zap.Logger{}, logger, "GetFromContext() should return a *zap.Logger when StreamInterceptor is in use")
	assert.Nil(d.t, err, "GetFromContext() should not return an error when StreamInterceptor is in use")
	logger.Debug("test")
	assert.Equal(d.t, 1, len(d.logs.All()), "there should be a log message")
	var okFields []string
	if len(d.logs.All()[0].Context) > 0 {
		for _, f := range d.logs.All()[0].Context {
			if f.Key == FieldRemoteAddr {
				assert.Equal(d.t, "bufconn", f.String, "remote addr has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "remote addr has a wrong type")
				okFields = append(okFields, FieldRemoteAddr)
			}
			if f.Key == FieldServerType {
				assert.Equal(d.t, "*zaplogger.dummyLoggerWithFields", f.String, "server type has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "server type has a wrong type")
				okFields = append(okFields, FieldServerType)
			}
			if f.Key == FieldMethod {
				assert.Equal(d.t, "/foobar.DummyService/FooS", f.String, "method has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "method has a wrong type")
				okFields = append(okFields, FieldMethod)
			}
		}
	}
	assert.Contains(d.t, okFields, FieldRemoteAddr, "expected remote addr field")
	assert.Contains(d.t, okFields, FieldMethod, "expected method field")
	for {
		_, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			d.t.Fatal(err)
		}
	}
	return nil
}

func TestWithFields(t *testing.T) {
	core, recordedLogs := observer.New(zapcore.DebugLevel)
	l, _ := New(
		WithLogger(zap.New(core)),
		WithFields(
			FieldServerType,
			FieldRemoteAddr,
			FieldRemoteAddr,
			FieldRemoteAddr,
		),
	)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(l.UnaryInterceptor()),
	}
	utils.TestCallFoo(t, &dummyLoggerWithFields{t: t, logs: recordedLogs}, nil, opts)

	core, recordedLogs = observer.New(zapcore.DebugLevel)
	l, _ = New(
		WithLogger(zap.New(core)),
		WithFields(
			FieldRemoteAddr,
			FieldMethod,
		),
	)
	opts = []grpc.ServerOption{
		grpc.StreamInterceptor(l.StreamInterceptor()),
	}
	utils.TestCallFooS(t, &dummyLoggerWithFields{t: t, logs: recordedLogs}, nil, opts)
}

type dummyLoggerWithServerName struct {
	foobar.UnimplementedDummyServiceServer
	t             *testing.T
	logs          *observer.ObservedLogs
	shouldBeFound bool
}

func (d *dummyLoggerWithServerName) Foo(ctx context.Context, in *foobar.Empty) (*foobar.Empty, error) {
	logger, err := GetFromContext(ctx)
	assert.IsType(d.t, &zap.Logger{}, logger, "GetFromContext() should return a *zap.Logger when UnaryInterceptor is in use")
	assert.Nil(d.t, err, "GetFromContext() should not return an error when UnaryInterceptor is in use")
	if logger == nil {
		return nil, nil
	}
	logger.Debug("test")
	assert.Equal(d.t, 1, len(d.logs.All()), "there should be a log message")
	found := false
	if len(d.logs.All()[0].Context) > 0 {
		for _, f := range d.logs.All()[0].Context {
			if f.Key == FieldServerName {
				assert.Equal(d.t, "hello world", f.String, "server name has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "server name has a wrong type")
				found = true
			}
		}
	}
	if d.shouldBeFound {
		assert.True(d.t, found, "server name should not have been found")
	} else {
		assert.False(d.t, found, "server name should have been found")
	}
	return &foobar.Empty{}, nil
}

func (d *dummyLoggerWithServerName) FooS(s foobar.DummyService_FooSServer) error {
	logger, err := GetFromContext(s.Context())
	assert.IsType(d.t, &zap.Logger{}, logger, "GetFromContext() should return a *zap.Logger when StreamInterceptor is in use")
	assert.Nil(d.t, err, "GetFromContext() should not return an error when StreamInterceptor is in use")
	if logger == nil {
		return nil
	}
	logger.Debug("test")
	assert.Equal(d.t, 1, len(d.logs.All()), "there should be a log message")
	found := false
	if len(d.logs.All()[0].Context) > 0 {
		for _, f := range d.logs.All()[0].Context {
			if f.Key == FieldServerName {
				assert.Equal(d.t, "hello world", f.String, "server name has a wrong value")
				assert.Equal(d.t, zapcore.StringType, f.Type, "server name has a wrong type")
				found = true
			}
		}
	}
	if d.shouldBeFound {
		assert.True(d.t, found, "server name should not have been found")
	} else {
		assert.False(d.t, found, "server name should have been found")
	}
	for {
		_, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			d.t.Fatal(err)
		}
	}
	return nil
}

func TestWithServerName(t *testing.T) {
	// Setting but not activating field => not found
	core, recordedLogs := observer.New(zapcore.DebugLevel)
	l, _ := New(
		WithLogger(zap.New(core)),
		WithServerName("hello world"),
	)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(l.UnaryInterceptor()),
	}
	utils.TestCallFoo(t, &dummyLoggerWithServerName{t: t, logs: recordedLogs, shouldBeFound: false}, nil, opts)

	// Not setting but activating field => not found
	core, recordedLogs = observer.New(zapcore.DebugLevel)
	l, _ = New(
		WithLogger(zap.New(core)),
		WithFields(FieldServerName),
	)
	opts = []grpc.ServerOption{
		grpc.StreamInterceptor(l.StreamInterceptor()),
	}
	utils.TestCallFooS(t, &dummyLoggerWithServerName{t: t, logs: recordedLogs, shouldBeFound: false}, nil, opts)

	// setting and activating field => found
	core, recordedLogs = observer.New(zapcore.DebugLevel)
	l, _ = New(
		WithLogger(zap.New(core)),
		WithServerName("hello world"),
		WithFields(FieldServerName),
	)
	opts = []grpc.ServerOption{
		grpc.StreamInterceptor(l.StreamInterceptor()),
	}
	utils.TestCallFooS(t, &dummyLoggerWithServerName{t: t, logs: recordedLogs, shouldBeFound: true}, nil, opts)

	// setting and activating field => found as stream
	core, recordedLogs = observer.New(zapcore.DebugLevel)
	l, _ = New(
		WithLogger(zap.New(core)),
		WithServerName("hello world"),
		WithFields(FieldServerName),
	)
	opts = []grpc.ServerOption{
		grpc.StreamInterceptor(l.StreamInterceptor()),
	}
	utils.TestCallFooS(t, &dummyLoggerWithServerName{t: t, logs: recordedLogs, shouldBeFound: true}, nil, opts)

	l, err := New(WithServerName(""))
	assert.ErrorIs(t, err, ErrInvalidOptionValue, "using an empty server name should return an ErrInvalidOptionValue")
	assert.Nil(t, l, "using an empty server name should return a nil logger")
}
