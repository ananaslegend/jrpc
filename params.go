package jrpc

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/valyala/fastjson"
)

type paramsKey struct{}

func Params(ctx context.Context) []byte {
	params := ctx.Value(paramsKey{})
	if params == nil {
		return nil
	}

	return params.([]byte)
}

func ParamsTo[T any](ctx context.Context) (*T, error) {
	params := Params(ctx)
	if params == nil {
		return nil, nil
	}

	t := new(T)

	if err := json.Unmarshal(params, t); err != nil {
		return nil, InvalidParamsError()
	}

	return t, nil
}

func setParams(ctx context.Context, reqValue *fastjson.Value) context.Context {
	var bts []byte

	if reqValue.Exists("params") {
		bts = reqValue.Get("params").MarshalTo(bts)
		if string(bts) == "null" {
			bts = nil
		}
	}

	return context.WithValue(ctx, paramsKey{}, bts)
}
