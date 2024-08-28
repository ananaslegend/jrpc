package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/ananaslegend/jrpc"
)

// all examples from official documentation (https://www.jsonrpc.org/specification) are implemented
// in the http_test.go and router_test.go, so here a simple example of using the http-router.
func main() {
	router := jrpc.NewHTTPRouter(
		":8080",
		jrpc.WithEndPoint("/jsonrpc"),
	)

	router.Method("ping", func(ctx context.Context) (any, error) {
		return "pong", nil
	})

	router.Method("ping.message", func(ctx context.Context) (any, error) {
		return t{"pong"}, nil
	})

	router.Method("params", func(ctx context.Context) (any, error) {
		p := personParams{}

		bts := jrpc.Params(ctx)

		if err := json.Unmarshal(bts, &p); err != nil {
			return nil, jrpc.InvalidParamsError()
		}

		return p, nil
	})

	router.Method("error", func(ctx context.Context) (any, error) {
		return nil, jrpc.InternalError("error")
	})

	router.Method("error.data", func(ctx context.Context) (any, error) {
		err := jrpc.InternalError()

		err.Data = map[string]interface{}{
			"key": "value",
		}

		return nil, err
	})

	router.Method("error.custom", func(ctx context.Context) (any, error) {
		err := errors.New("random error")

		return nil, err
	})

	router.Method("log", func(ctx context.Context) (any, error) {
		bts := jrpc.Params(ctx)

		var s string

		err := json.Unmarshal(bts, &s)
		if err != nil {
			log.Println(err)

			return nil, err
		}

		log.Println(s)

		return s, nil
	})

	router.Method("nil", func(ctx context.Context) (any, error) {
		return nil, nil
	})

	group := router.Group("group")

	group.Method("ping", func(ctx context.Context) (any, error) {
		return "group pong", nil
	})

	if err := router.Run(); err != nil {
		log.Fatal(err)
	}
}

type personParams struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type t struct {
	Message string `json:"message"`
}
