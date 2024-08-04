package jrpc

import (
	"context"
	"log/slog"
)

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

func (r *Router) Method(method string, handler func(ctx context.Context) (any, error)) {
	if r.path == "" {
		r.engine.handleMethod(method, handler)

		return
	}

	r.engine.handleMethod(r.path+"."+method, handler)
}
