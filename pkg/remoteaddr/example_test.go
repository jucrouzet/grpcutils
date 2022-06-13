package remoteaddr_test

import (
	"context"
	"fmt"

	"github.com/jucrouzet/grpcutils/pkg/remoteaddr"
)

// ExampleGetFromContext show how to get the remote address from client
func ExampleGetFromContext() {
	addr, err := remoteaddr.GetFromContext(ctx)

	if err != nil {
		// There was an error while getting the remote address
	}
	// addr is now client's remote address
	fmt.Println(addr)
}

var ctx context.Context
