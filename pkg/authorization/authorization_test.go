package authorization

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
)

func TestGetFromContext(t *testing.T) {
	ctx := context.TODO()
	err := GetFromContext(ctx, 5)
	assert.ErrorIs(t, err, ErrType, "GetFromContext() with a non pointer dest should return a ErrType error")

	val := 5
	valStr := ""
	err = GetFromContext(ctx, &val)
	assert.ErrorIs(t, err, ErrUnchecked, "GetFromContext() with a non pointer dest should return a ErrUnchecked error")

	ctxV := context.WithValue(ctx, contextValueKey, ErrMissing)
	err = GetFromContext(ctxV, &val)
	assert.ErrorIs(t, err, ErrMissing, "GetFromContext() with a known error in context value not set should return the error")
	ctxV = context.WithValue(ctx, contextValueKey, ErrInvalid)
	err = GetFromContext(ctxV, &val)
	assert.ErrorIs(t, err, ErrInvalid, "GetFromContext() with a known error in context value not set should return the error")

	ctxV = context.WithValue(ctx, contextValueKey, errors.New("foo bar"))
	err = GetFromContext(ctxV, &val)
	assert.ErrorIs(t, err, ErrInvalid, "GetFromContext() with an uknown error in context value not set should return a ErrInvalid")

	ctxV = context.WithValue(ctx, contextValueKey, 6)

	err = GetFromContext(ctxV, &valStr)
	assert.ErrorIs(t, err, ErrType, "GetFromContext() with an wrong type dest should return a ErrType error")

	err = GetFromContext(ctxV, &val)
	assert.Nil(t, err, "GetFromContext() with a valid auth value in context should not return an error")
	assert.Equal(t, 6, val, "GetFromContext() with a valid auth value should set the value in dest")

	type testType struct {
		a int
	}
	val2 := &testType{a: 42}
	var dest *testType
	ctxV = context.WithValue(ctx, contextValueKey, val2)
	err = GetFromContext(ctxV, &dest)
	assert.Nil(t, err, "GetFromContext() with a valid auth value in context should not return an error")
	assert.EqualValues(t, val2, dest, "GetFromContext() with a valid auth value should set the value in dest")
}

func TestWithMethodFunction(t *testing.T) {
	fooFunc := func(_ context.Context, credential string) (interface{}, error) {
		return nil, nil
	}
	a, err := New(
		WithMethodFunction("", fooFunc),
	)
	assert.Nil(t, a, "WithMethodFunction() should not return an Authorization with an invalid method")
	assert.ErrorIs(t, err, ErrInvalidOptionValue, "WithMethodFunction() should return a ErrInvalidOptionValue error with an invalid method")

	a, err = New(
		WithMethodFunction("test", nil),
	)
	assert.Nil(t, a, "WithMethodFunction() should not return an Authorization with an nil function")
	assert.ErrorIs(t, err, ErrInvalidOptionValue, "WithMethodFunction() should return a ErrInvalidOptionValue error with a nil function")
}

type dummyAuthorization struct {
	foobar.UnimplementedDummyServiceServer
	t               *testing.T
	expectingResult interface{}
	expectingError  error
}

func (d *dummyAuthorization) Foo(ctx context.Context, in *foobar.Empty) (*foobar.Empty, error) {
	var res string
	err := GetFromContext(ctx, &res)
	if d.expectingError != nil {
		assert.ErrorIs(d.t, err, d.expectingError, "expected the valid error for GetFromContext()")
		assert.Equal(d.t, "", res, "expected dest to be an empty string for GetFromContext()")
	} else if d.expectingResult != nil {
		assert.Nil(d.t, err, "expected error to be nil for GetFromContext()")
		assert.Equal(d.t, d.expectingResult, res, fmt.Sprintf("expected `%s` as result for GetFromContext()", d.expectingResult))
	}
	return &foobar.Empty{}, nil
}

func (d *dummyAuthorization) FooS(s foobar.DummyService_FooSServer) error {
	var res string
	err := GetFromContext(s.Context(), &res)
	if d.expectingError != nil {
		assert.ErrorIs(d.t, err, d.expectingError, "expected the valid error for GetFromContext()")
		assert.Equal(d.t, "", res, "expected dest to be an empty string for GetFromContext()")
	} else if d.expectingResult != nil {
		assert.Nil(d.t, err, "expected error to be nil for GetFromContext()")
		assert.Equal(d.t, d.expectingResult, res, fmt.Sprintf("expected `%s` as result for GetFromContext()", d.expectingResult))
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

func TestAuthorizationInterceptors(t *testing.T) {
	fooFunc := func(_ context.Context, credential string) (interface{}, error) {
		if credential == "bar" {
			return "ok", nil
		}
		return nil, errors.New("boo")
	}
	helloFunc := func(_ context.Context, credential string) (interface{}, error) {
		if credential == "world" {
			return "ok2", nil
		}
		return nil, errors.New("boo")
	}

	a, err := New(
		WithMethodFunction("foo", fooFunc),
		WithMethodFunction("hello", helloFunc),
	)
	assert.IsType(t, &Authorization{}, a, "New() should return an Authorization with valid options")
	assert.Nil(t, err, "New() should not return an error with a valid options")

	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingError: ErrUnchecked}, nil, nil)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingError: ErrUnchecked}, nil, nil)

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(a.GetUnaryInterceptor()),
		grpc.StreamInterceptor(a.GetStreamInterceptor()),
	}

	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingError: ErrMissing}, nil, opts)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingError: ErrMissing}, nil, opts)

	ctx, _ := AppendToOutgoingContext(context.TODO(), "hello", "world")
	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingResult: "ok2"}, nil, opts, ctx)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingResult: "ok2"}, nil, opts, ctx)

	ctx, _ = AppendToOutgoingContext(context.TODO(), "foo", "bar")
	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingResult: "ok"}, nil, opts, ctx)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingResult: "ok"}, nil, opts, ctx)

	ctx, _ = AppendToOutgoingContext(context.TODO(), "foo", "baz")
	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingError: ErrInvalid}, nil, opts, ctx)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingError: ErrInvalid}, nil, opts, ctx)

	ctx, _ = AppendToOutgoingContext(context.TODO(), "foo", "baz")
	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingError: ErrInvalid}, nil, opts, ctx)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingError: ErrInvalid}, nil, opts, ctx)

	ctx, _ = AppendToOutgoingContext(context.TODO(), "saycaptain", "say wut")
	utils.TestCallFoo(t, &dummyAuthorization{t: t, expectingError: ErrInvalid}, nil, opts, ctx)
	utils.TestCallFooS(t, &dummyAuthorization{t: t, expectingError: ErrInvalid}, nil, opts, ctx)
}

func TestAppendToOutgoingContext(t *testing.T) {
	ctx, err := AppendToOutgoingContext(context.TODO(), "", "42")
	assert.Nil(t, ctx, "SetAppendToOutgoingContext() with an invalid method should not return a context")
	assert.ErrorIs(t, err, ErrInvalidMethod, "SetAppendToOutgoingContext() with an invalid method should return a ErrInvalidMethod error with an invalid method")

	ctx, err = AppendToOutgoingContext(context.TODO(), "憑據", "this is a really weird 憑據 (it means credential according to google translate)")
	assert.Nil(t, err, "SetAppendToOutgoingContext() with a valid method should not return an error")
	assert.NotNil(t, ctx, "SetAppendToOutgoingContext() with a method should return a context")
	md, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok, "SetAppendToOutgoingContext() should return a context with metadatas")
	assert.Equal(t, 1, len(md.Get(MetadataName)), "SetAppendToOutgoingContext() should return a context with one MetadataName value")
	assert.Equal(t, "憑據 this is a really weird 憑據 (it means credential according to google translate)", md.Get(MetadataName)[0], "SetAppendToOutgoingContext() should return a context with the valid MetadataName value")

	ctx, err = AppendToOutgoingContext(ctx, "method2", "value2")
	assert.Nil(t, err, "SetAppendToOutgoingContext() with a valid method should not return an error")
	assert.NotNil(t, ctx, "SetAppendToOutgoingContext() with a method should return a context")
	md, ok = metadata.FromOutgoingContext(ctx)
	assert.True(t, ok, "SetAppendToOutgoingContext() should return a context with metadatas")
	assert.Equal(t, 1, len(md.Get(MetadataName)), "SetAppendToOutgoingContext() should return a context with one MetadataName value")
	assert.Equal(t, "method2 value2", md.Get(MetadataName)[0], "SetAppendToOutgoingContext() should return a context with the valid MetadataName value")
}

func Test_validateMethod(t *testing.T) {
	checkValid := func(method string) {
		assert.Nil(t, validateMethod(method), fmt.Sprintf(`validateMethod("%s") should not return an error`, strings.Replace(method, `"`, `\"`, -1)))
	}
	checkInvalid := func(method string) {
		assert.ErrorIs(t, validateMethod(method), ErrInvalidMethod, fmt.Sprintf(`validateMethod("%s") should not return a ErrInvalidMethod error`, strings.Replace(method, `"`, `\"`, -1)))
	}
	checkInvalid("")
	checkInvalid("  ")
	checkInvalid("Bearer")
	checkInvalid("见/見")
	checkInvalid("a-b")
	checkValid("a")
	checkValid("42")
	checkValid("bearer")
}

func TestAuthorization_parseMeta(t *testing.T) {
	var errOuch = errors.New("ouch")

	helloFunc := func(_ context.Context, credential string) (interface{}, error) {
		if credential == "world" {
			return "ok", nil
		}
		return "ok", errOuch
	}

	a, _ := New(
		WithMethodFunction("hello", helloFunc),
	)

	checkValid := func(expected string, authValue string) {
		md := metadata.MD{}
		md.Set(MetadataName, authValue)
		ctx := metadata.NewIncomingContext(context.TODO(), md)
		assert.Equal(t, expected, a.parseMeta(ctx), fmt.Sprintf(`parseMeta() should have returned "%s"`, strings.Replace(expected, `"`, `\"`, -1)))
	}
	checkInvalid := func(expectedErr error, authValue string) {
		ctx := context.TODO()
		if authValue != "" {
			md := metadata.MD{}
			md.Set(MetadataName, authValue)
			ctx = metadata.NewIncomingContext(ctx, md)
		}
		err, ok := a.parseMeta(ctx).(error)
		assert.True(t, ok, expectedErr, "expected parseMeta() to return an error")
		assert.ErrorIs(t, err, expectedErr, "expected parseMeta() to return the expected error")
	}

	checkValid("ok", "hello world")
	checkValid("ok", "hello   \t world")
	checkInvalid(errOuch, "hello monde")
	checkInvalid(ErrMissing, "")
	checkInvalid(ErrInvalid, "coucou")
	checkInvalid(ErrInvalidMethod, "heLLo monde")
	checkInvalid(ErrInvalid, "bearer token")
}
