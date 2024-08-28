package jrpc

import (
	"context"
	"log/slog"
)

type Option func(*handler)

func DontRender(h *handler) {
	h.dontRender = true
}

type Router struct {
	path   string
	engine *engine
}

func NewRouter(logger ...*slog.Logger) *Router {
	return &Router{
		engine: newEngine(logger...),
	}
}

func (r *Router) Group(method string) *Router {
	if r.path == "" {
		return &Router{
			path:   method,
			engine: r.engine,
		}
	}

	return &Router{
		path:   r.path + "." + method,
		engine: r.engine,
	}
}

func (r *Router) Method(method string, handlerFunc func(ctx context.Context) (any, error), opts ...Option) {
	h := &handler{handlerFunc: handlerFunc}

	for _, opt := range opts {
		opt(h)
	}

	if r.path == "" {
		r.engine.handleMethod(method, h)

		return
	}

	r.engine.handleMethod(r.path+"."+method, h)
}

func (r *Router) Handle(ctx context.Context, jsonRPCRequest []byte) []byte {
	return r.engine.handle(ctx, jsonRPCRequest)
}
