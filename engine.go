package jrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/goccy/go-json"
	"github.com/valyala/fastjson"
)

type engine struct {
	logger      *slog.Logger
	handlersMap map[string]func(ctx context.Context) (any, error)
}

func newEngine(logger ...*slog.Logger) *engine {
	r := &engine{
		handlersMap: make(map[string]func(ctx context.Context) (any, error)),
	}

	if len(logger) > 0 {
		r.logger = logger[0]

		return r
	}

	r.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	return r
}

func (router *engine) handleMethod(method string, handler func(ctx context.Context) (any, error)) {
	if _, ok := router.handlersMap[method]; ok {
		panic(fmt.Sprintf("method %s already exists", method))
	}

	router.handlersMap[method] = handler
}

func (router *engine) handle(ctx context.Context, bts []byte) []byte {
	arr, err := getRequestsArr(bts)
	if err != nil {
		return errorParsingJSONString
	}

	jobs, resultCh := workerPoolWithResult[*result](ctx, len(arr))

	for _, reqValue := range arr {
		jobs <- func() *result {
			id := getRequestID(reqValue)

			if !reqValue.Exists("method") {
				return &result{Err: InvalidRequestError(), Id: id}
			}

			method := string(reqValue.GetStringBytes("method"))

			ctx = setParams(ctx, reqValue)

			res, err := router.handleRequest(ctx, method)
			return processResult(id, err, res)
		}
	}

	close(jobs)

	resultList := make([]result, 0, len(arr))

	for res := range resultCh {
		if res != nil {
			resultList = append(resultList, *res)
		}
	}

	return renderResponse(resultList)
}

func getRequestsArr(body []byte) ([]*fastjson.Value, error) {
	var parser fastjson.Parser

	v, err := parser.ParseBytes(body)
	if err != nil {
		return nil, err
	}

	requestsArr, err := v.Array()
	if err != nil {
		return []*fastjson.Value{v}, nil
	}

	return requestsArr, nil
}

func processResult(id *ID, err error, res any) *result {
	if id != nil {
		if err != nil {
			var jrpcErr *Error
			if errors.As(err, &jrpcErr) {
				return &result{Err: jrpcErr, Id: id}
			}

			return &result{Err: InternalError(err.Error()), Id: id}
		}

		return &result{Err: nil, Res: res, Id: id}
	}

	return nil
}

func renderResponse(results []result) []byte {
	resWriter := &bytes.Buffer{}

	if len(results) > 1 {
		resWriter.WriteRune('[')
	}

	for i, res := range results {
		if i != 0 {
			resWriter.WriteRune(',')
		}

		resWriter.Write(res.RenderJSON())
	}

	if len(results) > 1 {
		resWriter.WriteRune(']')
	}

	return resWriter.Bytes()
}

func (router *engine) handleRequest(ctx context.Context, method string) (any, error) {
	handler, ok := router.handlersMap[method]
	if !ok {
		return nil, MethodNotFoundError()
	}

	return handler(ctx)
}

type result struct {
	Err *Error `json:"error"`
	Res any    `json:"result"`
	Id  *ID    `json:"id"`
}

func (r result) RenderJSON() []byte {
	if r.Err != nil {
		return []byte(fmt.Sprintf(`{"jsonrpc": "2.0", "error": %v, "id": %v}`, r.Err, r.Id))
	}

	resultJSON, err := json.Marshal(r.Res)
	if err != nil {
		return []byte(fmt.Sprintf(`{"jsonrpc": "2.0", "error": %v, "id": %v}`, InternalError("error during marshaling result: "+err.Error()), r.Id))
	}

	return []byte(fmt.Sprintf(`{"jsonrpc": "2.0", "result": %s, "id": %v}`, resultJSON, r.Id))
}
