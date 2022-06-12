package remoteaddr

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jucrouzet/grpcutils/internal/pkg/foobar"
	"github.com/jucrouzet/grpcutils/internal/pkg/utils"
)

type dummyRemote struct {
	foobar.UnimplementedDummyServiceServer
	t *testing.T
}

func (d *dummyRemote) Foo(ctx context.Context, in *foobar.Empty) (*foobar.Empty, error) {
	ip, err := GetFromContext(ctx)
	assert.NotNil(d.t, ip, "GetFromContext on a gRPC context should not return nil as ip")
	assert.Nil(d.t, err, "GetFromContext on a gRPC context should not return an error")
	assert.Equal(d.t, ip.String(), "bufconn", "GetFromContext on a gRPC context should return the valid address")
	return &foobar.Empty{}, nil
}

func (d *dummyRemote) FooS(s foobar.DummyService_FooSServer) error {
	for {
		_, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			d.t.Fatal(err)
		}
		ip, err := GetFromContext(s.Context())
		assert.NotNil(d.t, ip, "GetFromContext on a gRPC context should not return nil as ip")
		assert.Nil(d.t, err, "GetFromContext on a gRPC context should not return an error")
		assert.Equal(d.t, ip.String(), "bufconn", "GetFromContext on a gRPC context should return the valid address")
	}
	return nil
}

func TestGetFromContext(t *testing.T) {
	ip, err := GetFromContext(context.TODO())
	assert.Nil(t, ip, "GetFromContext on a non-gRPC context should return nil as ip")
	assert.ErrorIs(t, err, ErrNotAvailable, "GetFromContext on a non-gRPC context should return a ErrNotAvailable error")

	utils.TestCallFoo(t, &dummyRemote{t: t}, nil, nil)
	utils.TestCallFooS(t, &dummyRemote{t: t}, nil, nil)
}
