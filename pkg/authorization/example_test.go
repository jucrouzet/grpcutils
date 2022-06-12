package authorization_test

import (
	"context"

	"github.com/davecgh/go-spew/spew"

	"github.com/jucrouzet/grpcutils/pkg/authorization"
)

// ExampleNew creates a new buffconn gRPC server that checks authorization
func ExampleNew() {
	a, err := authorization.New(
		authorization.WithMethodFunction("bearer", checkToken),
	)
	if err != nil {
		panic(err)
	}
	spew.Dump(a)
}

func checkToken(ctx context.Context, token string) (interface{}, error) {
	return "ok", nil
}
