package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/ananaslegend/jrpc"
)

// all examples from official documentation (https://www.jsonrpc.org/specification) are implemented
// in the http_test.go and router_test.go, so here a simple example of using the router.
func main() {
	// create a new JSON RPC router
	router := jrpc.NewRouter()

	// Create a new JSON RPC method.
	// If you want to use JSON RPC router with request id (to logs, or whatever), but don`t need to give response
	// you can use jrpc.DontRender option and returning (nil, nil). It will skip rendering response part.
	// You can return any values, but it will be ignored.
	router.Method("Log", func(ctx context.Context) (any, error) {
		logParams := &LogParams{}

		// also you can use jrpc.ParamsTo[YourStruct](ctx), look at UpdateStatus method example.
		paramsBytes := jrpc.Params(ctx)
		err := json.Unmarshal(paramsBytes, logParams)
		if err != nil {
			return nil, jrpc.InvalidParamsError()
		}

		// Get request id from JSON RPC request.
		// RequestID can be string, int or float value.
		// It will return jrpc.NullRequestID if request id is not set.
		// If request does not contain requestID it calls "notification".
		// Notification is a request that does not require a response, so result will be nil.
		// Also, this request id contains in response, so in butch requests client can identify
		// result for each request.
		// If you dont need to give response to client, such as in our example with message broker,
		// you can use jrpc.DontRender option. It will skip rendering response part.
		reqID := jrpc.RequestID(ctx)

		log.Println(fmt.Sprintf("JSON RPC Log Handler Call, message: %s, reqID: %s", logParams.Message, reqID))

		return nil, nil
	}, jrpc.DontRender)

	// Create a new JSON RPC method group. It will handle all methods with prefix "Product."
	productRouter := router.Group("Product")

	productRouter.Method("UpdateStatus", func(ctx context.Context) (any, error) {
		// Get params from JSON RPC request
		param, err := jrpc.ParamsTo[UpdateProductStatusParam](ctx)
		if err != nil {
			// retry logic, logging or whatever

			return nil, jrpc.InvalidParamsError()
		}

		log.Println(fmt.Sprintf("%+v", param))

		if err = someLogic(param); err != nil {
			return nil, err
			// returning random err it will wrap into JSON RPC Internal error.
			// you can use jrpc.Error struct to create custom JSON RPC errors. Look at Error group method examples.
			// result: {"jsonrpc": "2.0", "error": {"code": -32603, "message": "product id is 0"}, "id": "560f3b56-38f8-4603-a27c-77d8cc2d2b4b"}
		}

		return true, nil
		// result: {"jsonrpc":"2.0","result":true,"id":"31e5739c-ee2b-44f0-bf9f-e38fc500479c"}
	})

	router.Method("Ping", func(ctx context.Context) (any, error) {
		return messageResult{Message: "pong"}, nil
		// result: {"jsonrpc":"2.0","result":{"message":"pong"},"id":123}
	})

	errRouter := router.Group("Error")

	errRouter.Method("Internal", func(ctx context.Context) (any, error) {
		return nil, jrpc.InternalError("error message")
		// result: {"jsonrpc":"2.0","error":{"code":-32603,"message":"error message"},"id":234}
	})

	errRouter.Method("InternalWithData", func(ctx context.Context) (any, error) {
		err := jrpc.InternalError()

		err.Data = map[string]interface{}{
			"key": "value",
		}

		return nil, err
		// result: {"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error","data":{"key":"value"}},"id":345}
	})

	errRouter.Method("Custom", func(ctx context.Context) (any, error) {
		err := &jrpc.Error{
			Code:    100,
			Message: "Custom Error",
		}

		return nil, err
		// result: {"jsonrpc":"2.0","error":{"code":100,"message":"Custom Error"},"id":456}
	})

	router.Method("Null", func(ctx context.Context) (any, error) {
		return nil, nil
		// result: {"jsonrpc":"2.0","result":null,"id":567}
	})

	// JSON RPC request consumer.
	msgBroker := newMessageBroker()

	// Just tests for the example.
	msgBroker.produceMessagesInJSONRPCFormat(examples)

	// Consume JSON RPC requests.
	for i := 0; i < len(examples); i++ {
		msg := msgBroker.getMessageInJSONRPCFormat()
		go func() {
			// result is result of handler function in JSON RPC format. If you use jrpc.DontRender option, it will be nil.
			result := router.Handle(context.Background(), msg)
			if result != nil {
				log.Println(fmt.Sprintf("JSON RPC Result: %s", string(result)))
			}

			msgBroker.wg.Done()
		}()
	}

	msgBroker.wg.Wait()
}

var (
	examples = [][]byte{
		[]byte(`{"jsonrpc":"2.0","method":"Log","params":{"msg":"hello from test"},"id":"42695bfa-2740-4c8e-b506-3650395c5f40"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Product.UpdateStatus","params":{"id":1,"status":"active"},"id":"31e5739c-ee2b-44f0-bf9f-e38fc500479c"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Product.UpdateStatus","params":{"id":0,"status":"active"},"id":"560f3b56-38f8-4603-a27c-77d8cc2d2b4b"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Ping","params":null,"id":"123"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Error.Internal","params":null,"id":"234"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Error.InternalWithData","params":null,"id":"345"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Error.Custom","params":null,"id":"456"}`),
		[]byte(`{"jsonrpc":"2.0","method":"Null","params":null,"id":"567"}`),
	}
)

type messageBroker struct {
	ch chan []byte
	wg *sync.WaitGroup
}

func newMessageBroker() *messageBroker {
	return &messageBroker{
		ch: make(chan []byte, 100),
		wg: &sync.WaitGroup{},
	}
}

func (d messageBroker) getMessageInJSONRPCFormat() []byte {
	return <-d.ch
}

func (d messageBroker) produceMessageInJSONRPCFormat(msg []byte) {
	d.ch <- msg
}

func (d messageBroker) produceMessagesInJSONRPCFormat(msgs [][]byte) {
	d.wg.Add(len(msgs))

	for _, msg := range msgs {
		d.ch <- msg
	}
}

type UpdateProductStatusParam struct {
	ProductID int    `json:"id"`
	Status    string `json:"status"`
}

type LogParams struct {
	Message string `json:"msg"`
}

func someLogic(param *UpdateProductStatusParam) error {
	if param.ProductID == 0 {
		return fmt.Errorf("product id is 0")
	}

	return nil
}

type messageResult struct {
	Message string `json:"message"`
}
