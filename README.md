# gRPCUtils : a collection of gRPC utilities for Go



Package holds some common utilities for gRPC development in go.

## Authorization

`authorization` handle call's authorization (authentification) by allowing client to send a credential
token with request in metadata.

Package can handle any authentification method as it is the user's responsability to validate a 
credential and returns associated data with this credential.

A common example would be a [JWT token](https://jwt.io/introduction), sent with the method `bearer` :

*Client :*

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/authorization"
    //...
)

func CallService(ctx context.Context) error {
    conn, err := grpc.DialContext(/*...*/)
	if err != nil {
        return err
	}
	client := grpcservice.NewServiceClient(conn)

    // Add the JWT token to call
    ctx, err = authorization.AppendToOutgoingContext(ctx, "bearer", "eyJhbGciO...")
    if err != nil {
        return err
    }
    v, err := client.ServiceMethod(ctx, &grpcservice.Value("blah"))
    // ...
}
```

*Server :*

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/authorization"
    //...
)

type User struct {
    FirstName string
    // ...
}


func InitServer(ctx context.Context) error {
    // Set interceptors while initializing gRPC server
	a, err := authorization.New(
		authorization.WithMethodFunction("bearer", checkToken),
	)
	server := grpc.NewServer(
		grpc.UnaryInterceptor(a.UnaryInterceptor()),
		grpc.StreamInterceptor(a.StreamInterceptor()),
	)
	grpcservice.RegisterDummyServiceServer(server, &myServer{})
}

// checkToken is responsible for checking the JWT token and return a user if valid
func checkToken(ctx context.Context, token string) (any, error) {
    // Parse JWT
    userID, err := parseJWT(credential)
    if err != nil {
        return nil, err
    }
    // Get user from JWT token's information
    user, err := getUserInDB(userID)
    if err != nil {
        return nil, err
    }
    // return it
    // the type of the returned variable should be the same when using GetFromContext
    return user, nil
}

// You can now use GetFromContext in gRPC method handlers
func (ms *myServer) MyUnaryMethod(ctx context.Context, param *grpcservice.Type) (*grpcservice.Type, error) {
   	var usr *User
	err := authorization.GetFromContext(ctx, &usr)

	if errors.Is(err, authorization.ErrMissing) {
        return nil, status.Errorf(codes.PermissionDenied, "method needs authorization")
	}
	if errors.Is(err, authorization.ErrInvalid) {
		return nil, status.Errorf(codes.PermissionDenied, "invalid credentials")
	}
    if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Errorf("failed checking credentials: %w", err)
	}
    // Do something with usr ...
}

func (ms *myServer) MyStreamMethod(s *grpcservice.Service_FooServer) error {
   	var usr *User
	err := authorization.GetFromContext(s.Context(), &usr)

	if errors.Is(err, authorization.ErrMissing) {
        return status.Errorf(codes.PermissionDenied, "method needs authorization")
	}
	if errors.Is(err, authorization.ErrInvalid) {
		return status.Errorf(codes.PermissionDenied, "invalid credentials")
	}
    if err != nil {
		return status.Errorf(codes.Internal, fmt.Errorf("failed checking credentials: %w", err)
	}
    // Do something with usr ...
}
```

## Remote address

`remoteaddr` is a simple wrapper to get client's remote address.

You can use it directly in gRPC method handlers :

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/remoteaddr"
    //...
)


func (ms *myServer) MyUnaryMethod(ctx context.Context, param *grpcservice.Type) (*grpcservice.Type, error) {
    addr, err := remoteaddr.GetFromContext(ctx)
    if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Errorf("failed checking remote address: %w", err)
	}
    fmt.Println(addr.String()) // "1.2.3.4:1234"
    // ...
}

func (ms *myServer) MyStreamMethod(s *grpcservice.Service_FooServer) error {
    addr, err := remoteaddr.GetFromContext(s.Context())
    if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Errorf("failed checking remote address: %w", err)
	}
    fmt.Println(addr.String()) // "1.2.3.4:1234"
    // ...
}
```
## Request correlation identifier

`requestid` handles a unique correlation identifier for each call like `X-Request-Id` for HTTP.

Client should sent a unique identifier for each request in outgoing context metadata :

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/requestid"
    //...
)

func CallService(ctx context.Context) error {
    conn, err := grpc.DialContext(/*...*/)
	if err != nil {
        return err
	}
	client := grpcservice.NewServiceClient(conn)

    // Add a request id to call
    ctx = requestid.AppendToOutgoingContext(ctx, "i'm an unique id")
    _, err := client.ServiceMethod(ctx, &grpcservice.Value("blah"))
    
    // Get request id from call's response
   	var header metadata.MD
    _, err := client.OtherMethod(ctxWithNoRequestID, &grpcservice.Value("blah"), grpc.Header(&header))
    if err != nil {
        panic(err)
    }
    rID := requestid.GetFromMeta(header)
}
```

On server side, `requestid` offers a `GetFromContext(ctx context.Context) string` method that returns
the call's request correlation identifier, if any (returns an empty string if not).

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/requestid"
    //...
)

func (ms *myServer) MyUnaryMethod(ctx context.Context, param *grpcservice.Type) (*grpcservice.Type, error) {
    requestID := requestid.GetFromContext(ctx)
    // ...
}
```

Using the provided interceptors will ensure that :

* a request correlation identifier is present in gRPC's IncomingContext's metadata, if not, one is generated and added
* request correlation identifiers (the one sent or the one generated) is sent in call's response header

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/requestid"
    //...
)

func InitServer(ctx context.Context) error {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(requestid.UnaryInterceptor),
		grpc.StreamInterceptor(requestid.StreamInterceptor),
	)
	grpcservice.RegisterDummyServiceServer(server, &myServer{})
    // ...
}
```

## Uber's zap logger for gRPC server handlers

`zaplogger` provides a way to implement Uber's [zap](https://github.com/uber-go/zap) logger in gRPC
server method handlers, with a set of automatically set fields for each message.

Provided fields are :

| Field  | Type | Description | Example  |
| :--- | :--- | :--- | :--- |
| `zaplogger.FieldServerName` | `zap.String` | Arbitrary value provided by `zaplogger.WithServerName()` in `New()` options  | `"my gRPC server"`  |
| `zaplogger.FieldServerType` | `zap.String` | Type (`fmt.Sprintf("%T")` result) or the gRPC server implentation struct  | `"*mypackage.MyGRPCServer"`  |
| `zaplogger.FieldRemoteAddr` | `zap.String` | Remote address (usually <ip>:<port>) of the client calling the method  |  `"127.0.0.1:1234"`  |
| `zaplogger.FieldMethod`     | `zap.String` | Name of the gRPC method called  | `"/package.Service/MyMethod"` |

Logger should be instanciated and added to interceptors like this :

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/zaplogger"
    //...
)

func InitServer(ctx context.Context) error {
	l, err := zaplogger.New(
        zaplogger.WithFields(
            zaplogger.FieldRemoteAddr,
            zaplogger.FieldServerName,
        ),
        zaplogger.WithServerName("my own server"),
    )
	server := grpc.NewServer(
		grpc.UnaryInterceptor(l.UnaryInterceptor()),
		grpc.StreamInterceptor(l.StreamInterceptor()),
	)
	grpcservice.RegisterDummyServiceServer(server, &myServer{})
    // ...
}
```

And can be used in method handlers like this :

```go
import (
    "context"
    // ...
    "github.com/jucrouzet/grpcutils/pkg/zaplogger"
    //...
)


func (ms *myServer) MyUnaryMethod(ctx context.Context, param *grpcservice.Type) (*grpcservice.Type, error) {
    logger, err := zaplogger.GetFromContext(ctx)
    if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Errorf("failed checking remote address: %w", err)
	}
    // ...
    logger.With(zap.Error(err)).Warn("failed doing something important")
    // ...
}

func (ms *myServer) MyStreamMethod(s *grpcservice.Service_FooServer) error {
    logger, err := zaplogger.GetFromContext(s.Context())
    if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Errorf("failed checking remote address: %w", err)
	}
    // ...
    logger.With(zap.Error(err)).Warn("failed doing something important")
    // ...
}
```
