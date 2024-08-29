JRPC - JSON-RPC 2.0 Go Router
---
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

## What is it?
JRPC is a JSON-RPC 2.0 router for Go. It allows you to easily create JSON-RPC 2.0 servers in gin-like style.
This lib implements HTTP transporting with `jrpc.HTTPRouter`, and you can implement your own transport with `jrcp.Router`. 
Look at the [examples folder](https://github.com/ananaslegend/jrpc/tree/main/examples) for more details.

JSON-RPC is a stateless, light-weight remote procedure call (RPC) protocol. It uses JSON (RFC 4627) as data format.

More information about JSON-RPC 2.0 can be found in [official JSON-RPC 2.0 Documntation](https://www.jsonrpc.org/specification).

## Installation
```bash
  go get -u github.com/ananaslegend/jrpc
```

## Examples
### Routers
#### HTTP Router
HTTP Router is a router that implements HTTP transport for JSON-RPC 2.0. You can set end point and logger by the options.
```go
router := jrpc.NewHTTPRouter(
    ":8080",
    jrpc.WithEndPoint("/jsonrpc"),
    jrpc.WithLogger(logger),
)
```

If you don't pass end-point, it will use `"/"` as end-point.
If you don't pass logger, it will use `slog.Default()` for logging.

#### General Router
General Router is a router that doesn't implement any transport. You can implement your own transport with this router.

Example with message broker:
```go
for {
    msg := msgBroker.getMessageInJSONRPCFormat()
    go func() {
        result := router.Handle(context.Background(), msg)
        if result != nil {
            log.Println(fmt.Sprintf("JSON RPC Result: %s", string(result)))
        }

        msgBroker.wg.Done()
    }()
}
```

### Handler
Handler is a function that processes the request and returns the result or error, and then it will be marshaled to JSON-RPC response.
Result is any type and at rendering it will be marshaled to JSON with json.Marshal, so better to add json tags.

Handler:
```go
router.Method("ping", func(ctx context.Context) (any, error) {
	return "pong", nil
})
```

Result:
```json
{"jsonrpc":"2.0","result":"pong","id": 1}
```

### Parsing request params
You can parse request params with `jrpc.ParamsTo[T any]` function with context.Context argument. It returns a pointer of passed type, or error.
```go
param, err := jrpc.ParamsTo[UpdateProductStatusParam](ctx)
if err != nil {
    return nil, jrpc.InvalidParamsError()
}
```

Also you can parse request params with `jrpc.Params` function with context.Context argument. It returns params as a byte slice.
```go
params := &RequestParams{}

paramsBytes := jrpc.Params(ctx)
err := json.Unmarshal(paramsBytes, params)
if err != nil {
    return nil, jrpc.InvalidParamsError()
}
```

### Returning Result
Result is any type and at rendering it will be marshaled to JSON with json.Marshal, so better to add json tags.

returning struct:
```go
router.Method("Ping", func(ctx context.Context) (any, error) {
    return messageResult{Message: "pong"}, nil
    // result: {"jsonrpc":"2.0","result":{"message":"pong"},"id":123}
})

type messageResult struct {
    Message string `json:"message"`
}
```

returning string:
```go
router.Method("ping", func(ctx context.Context) (any, error) {
	return "pong", nil
// result: {"jsonrpc":"2.0","result":"pong","id":567}
})
```

returning nil:
```go
router.Method("Null", func(ctx context.Context) (any, error) {
    return nil, nil
    // result: {"jsonrpc":"2.0","result":null,"id":567}
})
```

### Returning Error
JSON-RPC Errors.

Every JSON-RPC error has a code, message, and optional data fields.
You can read more about JSON-RPC errors in the [official JSON-RPC 2.0 Documntation](https://www.jsonrpc.org/specification#error_object).

You can return an error from your handlerFunc and it will wrap into `jrpc.Error` with code -32603 (Internal error) and message from error.Error() method.
```go
productRouter.Method("UpdateStatus", func(ctx context.Context) (any, error) {
    if err = someLogic(); err != nil {
        return nil, err
        // returning random err it will wrap into JSON RPC Internal error.
        // result: {"jsonrpc": "2.0", "error": {"code": -32603, "message": "product id is 0"}, "id": "560f3b56-38f8-4603-a27c-77d8cc2d2b4b"}
    }

    return true, nil
    // result: {"jsonrpc":"2.0","result":true,"id":"31e5739c-ee2b-44f0-bf9f-e38fc500479c"}
})
```

Standard errors. 
You can return standard errors with `jrpc` package functions such as `jrpc.InvalidRequestError`, `jrpc.InvalidParamsError`, `jrpc.ParseError`, `jrpc.MethodNotFoundError`, `jrpc.InternalError`
```go
errRouter.Method("Internal", func(ctx context.Context) (any, error) {
    return nil, jrpc.InternalError("error message")
    // result: {"jsonrpc":"2.0","error":{"code":-32603,"message":"error message"},"id":234}
})
```

All errors supports Data field, so you can add additional information to your error response:
```go
errRouter.Method("InternalWithData", func(ctx context.Context) (any, error) {
    err := jrpc.InternalError()

    err.Data = map[string]interface{}{
        "key": "value",
    }

    return nil, err
    // result: {"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error","data":{"key":"value"}},"id":345}
})
```

You can return a custom error with `jrpc.Error` function.
```go
errRouter.Method("Custom", func(ctx context.Context) (any, error) {
    err := &jrpc.Error{
        Code:    100,
        Message: "Custom Error",
    }

    return nil, err
    // result: {"jsonrpc":"2.0","error":{"code":100,"message":"Custom Error"},"id":456}
})
```

### Grouping
You can group your methods with `jrpc.Group` method. It will add a prefix to your method name.
```go
productRouter := router.Group("Product")

// it will be called as "Product.UpdateStatus"
productRouter.Method("UpdateStatus", func(ctx context.Context) (any, error) {
    return someLogic()
})
```

### Request ID
Request ID is a identifier for the request. It can be a string, number, float or null.
Requests without ID calls notifications, and they don't expect a response.
Clients can use request ID to match responses with requests in batch requests.

You can read more about Request ID and Notifications in the [official JSON-RPC 2.0 Documntation](https://www.jsonrpc.org/specification#request_object).

If you want to get request ID in your handler, you can use `jrpc.RequestID` function with context.Context argument.
It will return `jrpc.NullRequestID` if request id is not set.
```go
reqID := jrpc.RequestID(ctx)
```

### Options
If you want to use JSON RPC router with request id (to logs, or whatever), but don`t need to give response you can use jrpc.DontRender option and returning (nil, nil). 
It will skip rendering response part. You can return any values, but it will be ignored.

```go
router.Method("Ping", func(ctx context.Context) (any, error) {
    return nil, nil
}, jrpc.DontRender)
```