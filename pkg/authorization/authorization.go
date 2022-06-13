// Package authorization handles gRPC calls authorization via metadata, similary to the
// `Authorization` header (https://developer.mozilla.org/fr/docs/Web/HTTP/Headers/Authorization)
// in HTTP.
package authorization

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
)

const (
	// MetadataName is the name of the metadata that holds the authorization credential.
	MetadataName = "authorization"
)

var (
	// ErrInvalidOptionValue is returned when using an invalid option value
	ErrInvalidOptionValue = errors.New("invalid option value")
	// ErrUnchecked is returned when no Authorization interceptor are in use
	ErrUnchecked = errors.New("authorization metadata is unchecked")
	// ErrMissing is returned when request was made without authorization metadata
	ErrMissing = errors.New("request has no authorization credentials")
	// ErrInvalid is returned when request was made with an invalid authorization metadata
	ErrInvalid = errors.New("authorization credentials are invalid")
	// ErrType is returned when trying to get authorization result as the wrong type
	ErrType = errors.New("authorization has the wrong type")
	// ErrInvalidMethod is returned when trying to use an invalid authorization method
	ErrInvalidMethod = errors.New("invalid authorization method")
)

// Authorization handles authorization of methods via metadata
type Authorization struct {
	methods map[string]CredentialValidator
}

// Options is the Authorization option functions type
type Options func(*Authorization) error

// CredentialValidator is the function type for functions that validates credentials.
// The context passed is the unary or stream context, credential is the value/token provided by the caller.
// The value returned will be stored in the context and available in methods implementations by
// calling GetFromContext.
// The returned value type if correct must be the same as the one used in GetFromContext.
type CredentialValidator func(ctx context.Context, credential string) (any, error)

type contextValueKeyType string

var contextValueKey = contextValueKeyType("github.com/jucrouzet/grpcutils/authorization value")

// WithMethodFunction specifies a credential validation function to be used for a given authorization method.
// `method` must be a non empty string with lowercase alphanumeric characters, not containing whitespaces.
func WithMethodFunction(method string, fn CredentialValidator) Options {
	return func(a *Authorization) error {
		if fn == nil {
			return errors.New("cannot use a nil function")
		}
		if err := validateMethod(method); err != nil {
			return err
		}
		a.methods[method] = fn
		return nil
	}
}

// New creates a new instance of Authorization with specified options
func New(opts ...Options) (*Authorization, error) {
	a := &Authorization{
		methods: make(map[string]CredentialValidator),
	}
	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, fmt.Errorf("%w : %s", ErrInvalidOptionValue, err.Error())
		}
	}
	return a, nil
}

// GetFromContext gets the authorization value, if request has been made with a valid metadata
// and passed throught interceptor and sets ir into `dest`.
// If not, can return ErrUnchecked, ErrMissing, ErrInvalid or ErrType
func GetFromContext(ctx context.Context, dest interface{}) error {
	ddest := reflect.ValueOf(dest)
	if ddest.Kind() != reflect.Ptr {
		return fmt.Errorf("%w: dest must be a pointer", ErrType)
	}
	v := ctx.Value(contextValueKey)
	if v == nil {
		return ErrUnchecked
	}
	err, ok := v.(error)
	if ok {
		if errors.Is(err, ErrMissing) || errors.Is(err, ErrInvalid) {
			return err
		}
		return fmt.Errorf("%w: %s", ErrInvalid, err.Error())
	}
	from := reflect.ValueOf(v)
	if !ddest.Elem().Type().AssignableTo(from.Type()) {
		return fmt.Errorf("%w: expecting a %s but got value is a %s", ErrType, ddest.Elem().Type(), from.Type())
	}
	ddest.Elem().Set(from)
	return nil
}

// AppendToOutgoingContext will return a new context with authentification metadata for a given method
// appended to the outgoing context.
// `method` must be a non empty string with lowercase alphanumeric characters, not containing whitespaces.
// If context's outgoing metadata already contains a credential, it is replaced,
func AppendToOutgoingContext(ctx context.Context, method, credential string) (context.Context, error) {
	if err := validateMethod(method); err != nil {
		return nil, err
	}
	val := fmt.Sprintf("%s %s", strings.ToLower(method), credential)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	md.Set(MetadataName, val)
	return metadata.NewOutgoingContext(ctx, md), nil
}

// UnaryInterceptor returns a gRPC server unary interceptor that sets logger in call context
func (a *Authorization) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(context.WithValue(ctx, contextValueKey, a.parseMeta(ctx)), req)
	}
}

// StreamInterceptor returns a gRPC server stream interceptor that sets logger in stream context
func (a *Authorization) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ns := &utils.ServerStream{
			ServerStream: stream,
			Ctx:          context.WithValue(stream.Context(), contextValueKey, a.parseMeta(stream.Context())),
		}
		return handler(srv, ns)
	}
}

var authorizationMetaRegex = regexp.MustCompile(`(?m)^([^\s]+)\s+(.*)`)

func (a *Authorization) parseMeta(ctx context.Context) any {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md[MetadataName]) < 1 {
		return ErrMissing
	}
	res := authorizationMetaRegex.FindStringSubmatch(md[MetadataName][0])
	if res == nil {
		return fmt.Errorf("%w: invalid format for authorization metadata", ErrInvalid)
	}
	if err := validateMethod(res[1]); err != nil {
		return err
	}
	fn, ok := a.methods[res[1]]
	if !ok || fn == nil {
		return fmt.Errorf(`%w: authorization method "%s" is not supported`, ErrInvalid, res[1])
	}
	v, err := fn(ctx, res[2])
	if err != nil {
		return err
	}
	return v
}

func validateMethod(method string) error {
	if method == "" {
		return fmt.Errorf("%w: is empty", ErrInvalidMethod)
	}
	if strings.Contains(method, " ") {
		return fmt.Errorf("%w: must not contains whitespace", ErrInvalidMethod)
	}
	if method != strings.ToLower(method) {
		return fmt.Errorf("%w: must be lowercase", ErrInvalidMethod)
	}
	for _, r := range method {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("%w: must only contains alphanumeric characters", ErrInvalidMethod)
		}
	}
	return nil
}
