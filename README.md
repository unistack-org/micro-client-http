# HTTP Client
![Coverage](https://img.shields.io/badge/Coverage-42.6%25-yellow)

This plugin is an HTTP client for [Micro](https://pkg.go.dev/go.unistack.org/micro/v3).
It implements the [micro.Client](https://pkg.go.dev/go.unistack.org/micro/v3/client#Client) interface.

## Overview

The HTTP client wraps `net/http` to provide a robust client with service discovery, load balancing and 
implements HTTP rules defined in the [google/api/http.proto](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) specification.

## Limitations

* Streaming is not yet implemented.
* Only protobuf-generated messages are supported.

## Usage

```go
import (
    "go.unistack.org/micro/v3"
    http "go.unistack.org/micro-client-http/v3"
)

service := micro.NewService(
	micro.Name("my.service"),
	micro.Client(http.NewClient()),
)
```

### Simple call

```go
import (
    "go.unistack.org/micro/v3/client"
    http "go.unistack.org/micro-client-http/v3"
    jsoncodec "go.unistack.org/micro-codec-json/v3"
)

c := http.NewClient(
    client.Codec("application/json", jsoncodec.NewCodec()),
)

req := c.NewRequest(
    "user-service",
    "/user/{user_id}/order/{order_id}",
    &protoReq{UserId: "123", OrderId: 456},
)
rsp := new(protoRsp)

err := c.Call(
    ctx,
    req,
    rsp,
    client.WithAddress("example.com"),
)
```

### Call with specific options

```go
import (
    "go.unistack.org/micro/v3/client"
    http "go.unistack.org/micro-client-http/v3"
)

err := c.Call(
    ctx,
    req,
    rsp,
    client.WithAddress("example.com"),
    http.Method("POST"),
    http.Path("/user/{user_id}/order/{order_id}"),
    http.Body("*"), // <- use all fields from the proto request as HTTP request body or specify a single field name to use only that field (see Google API HTTP spec: google/api/http.proto)
)
```

### Call with request headers

```go
import (
    "go.unistack.org/micro/v3/metadata"
    http "go.unistack.org/micro-client-http/v3"
)

ctx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
    "Authorization", "Bearer token",
    "My-Header", "My-Header-Value",
))

err := c.Call(
    ctx,
    req,
    rsp,
    http.Header("Authorization", "true", "My-Header", "false"), // <- call option that declares required/optional headers
)
```

### Call with response headers

```go
import (
    "go.unistack.org/micro/v3/metadata"
    http "go.unistack.org/micro-client-http/v3"
)

respMetadata := metadata.Metadata{}

err := c.Call(
    ctx,
    req,
    rsp,
    client.WithResponseMetadata(&respMetadata), // <- metadata with response headers
)
```

### Call with cookies

```go
import (
    "go.unistack.org/micro/v3/metadata"
    http "go.unistack.org/micro-client-http/v3"
)

ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
    "Cookie", "session_id=abc123; theme=dark",
))

err := c.Call(
    ctx,
    req,
    rsp,
    http.Cookie("session_id", "true", "theme", "false"), // <- call option that declares required/optional cookies
)
```

### Call with error mapping

```go
import (
    http "go.unistack.org/micro-client-http/v3"
    status "go.unistack.org/micro-client-http/v3/status"
    jsoncodec "go.unistack.org/micro-codec-json/v3"
)

err := c.Call(
    ctx,
    req,
    rsp,
    http.ErrorMap(map[string]any{
        "default": &protoDefaultError{}, // <- default case
        "403":     &protoSpecialError{}, // <- key is the HTTP status code that is mapped to this error
    }),
)

if err != nil {
    s, ok := status.FromError(err)
    if !ok {...}
    
    code    := s.Code()     // HTTP status code
    message := s.Message()  // HTTP status text
    details := s.Details()  // Error type mapped from ErrorMap
}
```
