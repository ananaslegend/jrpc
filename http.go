package jrpc

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type Option func(*HTTPRouter)

type HTTPRouter struct {
	srv *http.Server

	endPoint string
	logger   *slog.Logger

	*Router
}

func NewHTTPRouter(addr string, opts ...Option) *HTTPRouter {
	router := &HTTPRouter{
		srv: &http.Server{
			Addr: addr,
		},
	}

	for _, opt := range opts {
		opt(router)
	}

	if router.logger == nil {
		router.logger = slog.Default()
	}

	router.Router = NewRouter(router.logger)

	if router.endPoint == "" {
		router.endPoint = "/"
	}

	return router
}

func WithLogger(logger *slog.Logger) Option {
	return func(router *HTTPRouter) {
		router.logger = logger
	}
}

func WithEndPoint(endPoint string) Option {
	return func(router *HTTPRouter) {
		router.endPoint = endPoint
	}
}

func (httpRouter *HTTPRouter) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc(httpRouter.endPoint, func(w http.ResponseWriter, req *http.Request) {
		if req.RequestURI != httpRouter.endPoint {
			http.Error(w, "404 page not found", http.StatusNotFound)

			return
		}

		httpRouter.handle(w, req)
	})

	httpRouter.srv.Handler = mux

	return httpRouter.srv.ListenAndServe()
}

func (httpRouter *HTTPRouter) Close() error {
	return httpRouter.srv.Close()
}

func (httpRouter *HTTPRouter) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bts, err := io.ReadAll(r.Body)
	if err != nil {
		_, err = w.Write(errorParsingJSONString)
		if err != nil {
			httpRouter.logger.Error(fmt.Sprintf("error during write into ResponseWriter: %v", err.Error()))

			return
		}
	}

	res := httpRouter.Router.engine.handle(r.Context(), bts)

	_, err = w.Write(res)
	if err != nil {
		httpRouter.logger.Error(fmt.Sprintf("error during write into ResponseWriter: %v", err.Error()))

		return
	}
}
