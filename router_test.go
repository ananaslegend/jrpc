package jrpc_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/goccy/go-json"

	"github.com/ananaslegend/jrpc"
)

type handler struct {
	method      string
	handlerFunc func(ctx context.Context) (any, error)
}

var (
	subtractHandler = handler{
		method: "subtract",
		handlerFunc: func(ctx context.Context) (any, error) {
			params := jrpc.Params(ctx)

			var p [2]int
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, jrpc.InvalidParamsError()
			}

			return p[0] - p[1], nil
		},
	}

	namedSubtractHandler = handler{
		method: "subtract",
		handlerFunc: func(ctx context.Context) (any, error) {
			params, err := jrpc.ParamsTo[struct {
				Subtrahend int `json:"subtrahend"`
				Minuend    int `json:"minuend"`
			}](ctx)
			if err != nil {
				return nil, err
			}

			return params.Minuend - params.Subtrahend, nil
		},
	}

	notificationHandler = handler{
		method: "update",
		handlerFunc: func(ctx context.Context) (any, error) {
			return nil, nil
		},
	}

	sumHandler = handler{
		method: "sum",
		handlerFunc: func(ctx context.Context) (any, error) {
			params := jrpc.Params(ctx)

			var p [3]int
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, jrpc.InvalidParamsError()
			}

			return p[0] + p[1] + p[2], nil
		},
	}

	notificationHelloHandler = handler{
		method: "notify_hello",
		handlerFunc: func(ctx context.Context) (any, error) {
			return nil, nil
		},
	}

	getDataHandler = handler{
		method: "get_data",
		handlerFunc: func(ctx context.Context) (any, error) {
			return []any{"hello", 5}, nil
		},
	}
)

// examples from https://www.jsonrpc.org/specification
func Test_JSON_RPC_2_0_Specification_Examples(t *testing.T) {
	tests := []struct {
		name        string
		rpcHandlers []handler
		request     []byte
		result      []byte
	}{
		{
			name:        "rpc call with positional parameters",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "subtract", "params": [42, 23], "id": 1}`),
			result:      []byte(`{"jsonrpc": "2.0", "result": 19, "id": 1}`),
		},
		{
			name:        "rpc call with positional parameters",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "subtract", "params": [23, 42], "id": 2}`),
			result:      []byte(`{"jsonrpc": "2.0", "result": -19, "id": 2}`),
		},
		{
			name:        "rpc call with named parameters",
			rpcHandlers: []handler{namedSubtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}`),
			result:      []byte(`{"jsonrpc": "2.0", "result": 19, "id": 3}`),
		},
		{
			name:        "rpc call with named parameters",
			rpcHandlers: []handler{namedSubtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "subtract", "params": {"minuend": 42, "subtrahend": 23}, "id": 4}`),
			result:      []byte(`{"jsonrpc": "2.0", "result": 19, "id": 4}`),
		},
		{
			name:        "a notification",
			rpcHandlers: []handler{notificationHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "update", "params": [1,2,3,4,5]}`),
			result:      []byte(``),
		},
		{
			name:        "a notification",
			rpcHandlers: []handler{notificationHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "foobar"}`),
			result:      []byte(``),
		},
		{
			name:        "rpc call of non-existent method",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "foobar", "id": "1"}`),
			result:      []byte(`{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "1"}`),
		},
		{
			name:        "rpc call with invalid JSON",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`),
			result:      []byte(`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`),
		},
		{
			name:        "rpc call with invalid Request object",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`{"jsonrpc": "2.0", "method": 1, "params": "bar"}`),
			result:      []byte(`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`),
		},
		{
			name:        "rpc call Batch, invalid JSON",
			rpcHandlers: []handler{subtractHandler},
			request: []byte(`[
				{"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
			  	{"jsonrpc": "2.0", "method"
			]`),
			result: []byte(`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`),
		},
		{
			name:        "rpc call with invalid Request object",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`[]`),
			result:      []byte(`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`),
		},
		{
			name:        "rpc call with an invalid Batch (but not empty)",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`[1]`),
			result: []byte(`[
  				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
			]`),
		},
		{
			name:        "rpc call with invalid Batch",
			rpcHandlers: []handler{subtractHandler},
			request:     []byte(`[1,2,3]`),
			result: []byte(`[
				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
			  	{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
			  	{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
			]`),
		},
		{
			name:        "rpc call with invalid Request object",
			rpcHandlers: []handler{sumHandler, notificationHelloHandler, subtractHandler, getDataHandler},
			request: []byte(`[
				{"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
				{"jsonrpc": "2.0", "method": "notify_hello", "params": [7]},
				{"jsonrpc": "2.0", "method": "subtract", "params": [42,23], "id": "2"},
				{"foo": "boo"},
				{"jsonrpc": "2.0", "method": "foo.get", "params": {"name": "myself"}, "id": "5"},
				{"jsonrpc": "2.0", "method": "get_data", "id": "9"} 
			]`),
			result: []byte(`[
				{"jsonrpc": "2.0", "result": 7, "id": "1"},
				{"jsonrpc": "2.0", "result": 19, "id": "2"},
				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
				{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "5"},
				{"jsonrpc": "2.0", "result": ["hello", 5], "id": "9"}
			]`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := jrpc.NewRouter()

			for _, h := range tt.rpcHandlers {
				router.Method(h.method, h.handlerFunc)
			}

			result := router.Handle(context.Background(), tt.request)

			equals, err := resultsEquals(string(result), string(tt.result))
			if err != nil {
				t.Errorf("error comparing results: %s", err.Error())
			}

			if !equals {
				t.Errorf("got %s, want %s", string(result), string(tt.result))
			}
		})
	}
}

func resultsEquals(actual, expected string) (bool, error) {
	if actual == expected {
		return true, nil
	}

	var o1 interface{}
	var o2 interface{}

	err := json.Unmarshal([]byte(actual), &o1)
	if err != nil {
		return false, fmt.Errorf("error unmarshalling actual result string: %s", err.Error())
	}

	err = json.Unmarshal([]byte(expected), &o2)
	if err != nil {
		return false, fmt.Errorf("error unmarshalling expected result string: %s", err.Error())
	}

	o1 = sortJSONData(o1)
	o2 = sortJSONData(o2)

	return reflect.DeepEqual(o1, o2), nil
}

func sortJSONData(data interface{}) interface{} {
	switch v := data.(type) {
	case []interface{}:
		for i := range v {
			v[i] = sortJSONData(v[i])
		}
		sort.SliceStable(v, func(i, j int) bool {
			return fmt.Sprintf("%v", v[i]) < fmt.Sprintf("%v", v[j])
		})
		return v
	case map[string]interface{}:
		for key := range v {
			v[key] = sortJSONData(v[key])
		}
		return v
	default:
		return v
	}
}
