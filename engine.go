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

type handler struct {
	handlerFunc func(ctx context.Context) (any, error)
	dontRender  bool
}

type engine struct {
	handlersMap map[string]*handler

	logger          *slog.Logger
	logRequestFunc  func(req []byte, logger *slog.Logger)
	logNotFoundFunc func(method string, logger *slog.Logger)
}

func (router *engine) logRequest(req []byte) {
	if router.logRequestFunc != nil {
		router.logRequestFunc(req, router.logger)
	}
}

func (router *engine) logNotFound(method string) {
	if router.logNotFoundFunc != nil {
		router.logNotFoundFunc(method, router.logger)
	}
}

func newEngine(logger ...*slog.Logger) *engine {
	r := &engine{
		handlersMap: make(map[string]*handler),
	}

	if len(logger) > 0 {
		r.logger = logger[0]

		return r
	}

	r.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	return r
}

func (router *engine) handleMethod(method string, h *handler) {
	if _, ok := router.handlersMap[method]; ok {
		panic(fmt.Sprintf("method %s already exists", method))
	}

	router.handlersMap[method] = h
}

func (router *engine) handle(ctx context.Context, bts []byte) []byte {
	router.logRequest(bts)

	arr, isButch, err := getRequestsArr(bts)
	if err != nil {
		return errorParsingJSONString
	}

	if len(arr) == 0 {
		return errorInvalidRequest
	}

	jobs, resultCh := workerPoolWithResult[*result](ctx, len(arr))

	for _, reqValue := range arr {
		jobs <- func() *result {
			id := getRequestID(reqValue)

			ctx = setRequestID(ctx, id)

			if !reqValue.Exists("method") {
				id.renderNull = true
				return &result{Err: InvalidRequestError(), Id: id}
			}

			method := string(reqValue.GetStringBytes("method"))
			if method == "" {
				return &result{Err: InvalidRequestError(), Id: id}
			}

			ctx = setParams(ctx, reqValue)

			h, ok := router.handlersMap[method]
			if !ok {
				router.logNotFound(method)

				return processResult(id, MethodNotFoundError(), nil)
			}

			if h.dontRender || id == nil {
				go h.handlerFunc(ctx)

				return nil
			}

			res, err := h.handlerFunc(ctx)

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

	return renderResponse(resultList, isButch)
}

func getRequestsArr(body []byte) ([]*fastjson.Value, bool, error) {
	var parser fastjson.Parser

	v, err := parser.ParseBytes(body)
	if err != nil {
		return nil, false, err
	}

	requestsArr, err := v.Array()
	if err != nil {
		return []*fastjson.Value{v}, false, nil
	}

	return requestsArr, true, nil
}

func processResult(id *requestID, err error, res any) *result {
	if id != nil {
		if !id.notNull && !id.renderNull { // todo
			return nil
		}

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

func renderResponse(results []result, isButch bool) []byte {
	if len(results) == 0 {
		return nil
	}

	resWriter := &bytes.Buffer{}

	if isButch {
		resWriter.WriteRune('[')
	}

	var firstRendered bool

	for _, res := range results {
		if firstRendered {
			resWriter.WriteRune(',')
		}

		resWriter.Write(res.RenderJSON())
		firstRendered = true

	}

	if isButch {
		resWriter.WriteRune(']')
	}

	return resWriter.Bytes()
}

func (router *engine) handleRequest(ctx context.Context, method string) (any, error) {
	h, ok := router.handlersMap[method]
	if !ok {
		return nil, MethodNotFoundError()
	}

	return h.handlerFunc(ctx)
}

type result struct {
	Err *Error     `json:"error"`
	Res any        `json:"result"`
	Id  *requestID `json:"id"`
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
