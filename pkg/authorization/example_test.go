package authorization_test

import (
	"context"
	"errors"

	"google.golang.org/grpc"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/pkg/authorization"
)

// ExampleNew creates a new gRPC server that checks authorization with the `basic` method
func ExampleNew() {
	// checkToken is a function that parses token, see the `WithMethodFunction` example
	a, err := authorization.New(
		authorization.WithMethodFunction("basic", checkToken),
	)
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer(
		grpc.UnaryInterceptor(a.UnaryInterceptor()),
		grpc.StreamInterceptor(a.StreamInterceptor()),
	)
	foobar.RegisterDummyServiceServer(server, &foobar.UnimplementedDummyServiceServer{})
}

// ExampleWithMethodFunction show how to define a `CredentialValidator` function that parses credentials
func ExampleWithMethodFunction() {
	var credentialValidator authorization.CredentialValidator

	credentialValidator = func(ctx context.Context, credential string) (any, error) {
		userID, err := parseJWT(credential)
		if err != nil {
			return nil, err
		}
		user, err := getUserInDB(userID)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	// auth'd user can later be get (if valid token) with :
	// var usr *User
	// err := authorization.GetFromContext(ctx, &usr)
	authorization.New(
		authorization.WithMethodFunction("bearer", credentialValidator),
	)
}

// ExampleGetFromContext show how to get the `CredentialValidator` result in a gRPC method handler
func ExampleGetFromContext() {
	var usr *User
	err := authorization.GetFromContext(ctx, &usr)

	if errors.Is(err, authorization.ErrMissing) {
		// No authorization was sent from client
	}
	if errors.Is(err, authorization.ErrInvalid) {
		// Authorization sent from client is invalid
	}
	if err != nil {
		// Another error (like wrong type or interceptor not used)
	}
	// usr is now the result of CredentialValidator is it returned a *User

}

func checkToken(ctx context.Context, token string) (any, error) {
	return "", nil
}

func parseJWT(tok string) (string, error) {
	return "", nil
}

func getUserInDB(userID string) (*User, error) {
	return nil, nil
}

var ctx context.Context

type User struct {
}
