package jrpc_test

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/ananaslegend/jrpc"
)

// examples from https://www.jsonrpc.org/specification
func Test_HTTP_JSON_RPC_2_0_Specification_Examples(t *testing.T) {
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
			router := jrpc.NewHTTPRouter(":8080")

			for _, h := range tt.rpcHandlers {
				router.Method(h.method, h.handlerFunc)
			}

			r := httptest.NewRequest("POST", "/", bytes.NewReader(tt.request))
			w := httptest.NewRecorder()

			router.Handle(w, r)

			resp := w.Result()
			defer resp.Body.Close()

			bts, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != 200 {
				t.Errorf("got %d, want 200", resp.StatusCode)
			}

			equals, err := resultsEquals(string(bts), string(tt.result))
			if err != nil {
				t.Errorf("error comparing results: %s", err.Error())
			}

			if !equals {
				t.Errorf("got %s, want %s", string(bts), string(tt.result))
			}
		})
	}
}
