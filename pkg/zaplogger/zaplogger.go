// Package zaplogger allows to set and use a go.uber.org/zap logger in gRPC service handlers
package zaplogger

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
	"github.com/jucrouzet/grpcutils/pkg/remoteaddr"
	"github.com/jucrouzet/grpcutils/pkg/requestid"
)

// Logger is a uber/zap logger for a grpc server methods
type Logger struct {
	fields     []Field
	logger     *zap.Logger
	serverName string
}

// Option is the Logger option functions type
type Option func(*Logger) error

// Field is the type of logger fields
type Field string

const (
	// FieldServerName adds the server name in log messages
	FieldServerName = "server_name"
	// FieldServerType adds the server type in log messages
	FieldServerType = "server_type"
	// FieldRemoteAddr adds the remote address of the caller in log messages
	FieldRemoteAddr = "remote_addr"
	// FieldMethod adds the called method name in log messages
	FieldMethod = "method"
	// FieldRequestID adds the request unique correlation ID in log messages
	// See github.com/jucrouzet/grpcutils/pkg/requestid
	FieldRequestID = "requestid"
)

var (
	// ErrInvalidOptionValue is returned when trying to use an invalid option value
	ErrInvalidOptionValue = errors.New("invalid option value")
	// ErrNoLoggerInContext is returned when trying get a logger from a context that doesn't have one
	ErrNoLoggerInContext = errors.New("no logger in context")
)

type contextValueKeyType string

var contextValueKey = contextValueKeyType("github.com/jucrouzet/grpcutils/zaplogger value")

// GetFromContext returns the logger from a context that has been set in UnaryInterceptor or
// StreamInterceptor.
// If logger is not set in context and noopLoggerIfNotPresent is not specified or false
// ErrNoLoggerInContext is returned.
// If logger is not set in context and noopLoggerIfNotPresent is true, a noop logger is returned
// and err can be ignored.
func GetFromContext(ctx context.Context, noopLoggerIfNotPresent ...bool) (*zap.Logger, error) {
	logger, ok := ctx.Value(contextValueKey).(*zap.Logger)
	if !ok || logger == nil {
		if len(noopLoggerIfNotPresent) > 0 && noopLoggerIfNotPresent[0] {
			return zap.New(nil), nil
		}
		return nil, ErrNoLoggerInContext
	}
	return logger, nil
}

// WithLogger specifies which uber/zap instance to use for logging.
// If not set, a new Development logger will be created.
func WithLogger(logger *zap.Logger) Option {
	return func(l *Logger) error {
		if logger == nil {
			return errors.New("cannot use a nil logger")
		}
		l.logger = logger
		return nil
	}
}

// WithFields adds fields to log messages
func WithFields(fields ...Field) Option {
	return func(l *Logger) error {
		for _, f := range fields {
			l.addField(f)
		}
		return nil
	}
}

// WithServerName sets the FieldServerName field value to log message.
func WithServerName(serverName string) Option {
	return func(l *Logger) error {
		if serverName == "" {
			return errors.New("server name cannot be empty")
		}
		l.serverName = serverName
		return nil
	}
}

// New creates a new instance of Logger with specified options
func New(opts ...Option) (*Logger, error) {
	l := &Logger{}
	for _, opt := range opts {
		if err := opt(l); err != nil {
			return nil, fmt.Errorf("%w : %s", ErrInvalidOptionValue, err.Error())
		}
	}
	if l.logger == nil {
		ll, err := zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("failed creating a new logger: %w", err)
		}
		l.logger = ll
	}
	if l.hasField(FieldServerName) && l.serverName != "" {
		l.logger = l.logger.With(zap.String(FieldServerName, l.serverName))
	}
	return l, nil
}

// GetLogger returns the zap logger
func (l *Logger) GetLogger() *zap.Logger {
	return l.logger
}

// UnaryInterceptor returns a gRPC server unary interceptor that sets logger in call context
func (l *Logger) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		infos *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		logger, err := l.getForUnary(ctx, infos)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed setting request logger")
		}
		return handler(context.WithValue(ctx, contextValueKey, logger), req)
	}
}

// StreamInterceptor returns a gRPC server stream interceptor that sets logger in stream context
func (l *Logger) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		infos *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger, err := l.getForStream(stream.Context(), infos, srv)
		if err != nil {
			return status.Error(codes.Internal, "failed setting request logger")
		}
		ns := &utils.ServerStream{
			ServerStream: stream,
			Ctx:          context.WithValue(stream.Context(), contextValueKey, logger),
		}
		return handler(srv, ns)
	}
}

func (l *Logger) addField(field Field) {
	for _, f := range l.fields {
		if f == field {
			return
		}
	}
	l.fields = append(l.fields, field)
}

func (l *Logger) hasField(field Field) bool {
	for _, f := range l.fields {
		if f == field {
			return true
		}
	}
	return false
}

func (l *Logger) getForUnary(ctx context.Context, infos *grpc.UnaryServerInfo) (*zap.Logger, error) {
	logger := l.GetLogger()
	if l.hasField(FieldRemoteAddr) {
		addr, err := remoteaddr.GetFromContext(ctx)
		if err != nil {
			return nil, err
		}
		logger = logger.With(zap.String(FieldRemoteAddr, addr.String()))
	}
	if l.hasField(FieldMethod) {
		logger = logger.With(zap.String(FieldMethod, infos.FullMethod))
	}
	if l.hasField(FieldServerType) {
		logger = logger.With(zap.String(FieldServerType, fmt.Sprintf("%T", infos.Server)))
	}
	if l.hasField(FieldRequestID) && requestid.GetFromContext(ctx) != "" {
		logger = logger.With(zap.String(FieldRequestID, requestid.GetFromContext(ctx)))
	}
	return logger, nil
}

func (l *Logger) getForStream(
	ctx context.Context,
	infos *grpc.StreamServerInfo,
	srv interface{},
) (*zap.Logger, error) {
	logger := l.GetLogger()
	if l.hasField(FieldRemoteAddr) {
		addr, err := remoteaddr.GetFromContext(ctx)
		if err != nil {
			return nil, err
		}
		logger = logger.With(zap.String(FieldRemoteAddr, addr.String()))
	}
	if l.hasField(FieldMethod) {
		logger = logger.With(zap.String(FieldMethod, infos.FullMethod))
	}
	if l.hasField(FieldServerType) {
		logger = logger.With(zap.String(FieldServerType, fmt.Sprintf("%T", srv)))
	}
	if l.hasField(FieldRequestID) && requestid.GetFromContext(ctx) != "" {
		logger = logger.With(zap.String(FieldRequestID, requestid.GetFromContext(ctx)))
	}
	return logger, nil
}
