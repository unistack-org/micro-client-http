# HTTP Client
![Coverage](https://img.shields.io/badge/Coverage-20.8%25-red)

This plugin is a http client for micro.

## Overview

The http client wraps `net/http` to provide a robust micro client with service discovery, load balancing and streaming. 
It complies with the [micro.Client](https://godoc.org/go.unistack.org/micro-client-http/v3#Client) interface.

## Usage

### Use directly

```go
import "go.unistack.org/micro-client-http/v3"

service := micro.NewService(
	micro.Name("my.service"),
	micro.Client(http.NewClient()),
)
```

### Call Service

Assuming you have a http service "my.service" with path "/foo/bar"
```go
// new client
client := http.NewClient()

// create request/response
request := client.NewRequest("my.service", "/foo/bar", protoRequest{})
response := new(protoResponse)

// call service
err := client.Call(context.TODO(), request, response)
```

or you can call any rest api or site and unmarshal to response struct
```go
// new client
client := client.NewClientCallOptions(http.NewClient(), http.Address("https://api.github.com"))

req := client.NewRequest("github", "/users/vtolstov", nil)
rsp := make(map[string]interface{})

err := c.Call(context.TODO(), req, &rsp, mhttp.Method(http.MethodGet)) 
```

Look at http_test.go for detailed use.

### Encoding

Default protobuf with content-type application/proto
```go
client.NewRequest("service", "/path", protoRequest{})
```

Json with content-type application/json
```go
client.NewJsonRequest("service", "/path", jsonRequest{})
```
