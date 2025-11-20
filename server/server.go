package server

import (
	"context"
	"net/http"

	"github.com/omalloc/tavern/contrib/log"
	"github.com/omalloc/tavern/contrib/transport"
)

type HTTPServer struct {
	*http.Server
}

func NewServer() transport.Server {
	return &HTTPServer{}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.Server = &http.Server{}
	return s.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}

func (s *HTTPServer) buildHandler(tripper http.RoundTripper) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var clog = log.Context(req.Context())
		var resp *http.Response
		var err error

		// finally close response body
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		resp, err = tripper.RoundTrip(req)
		if err != nil {
			clog.Errorf("request %s %s failed: %s", req.Method, req.URL.Path, err)
		}

		doCopyBody := func() {
			if resp.Body == nil {
				return
			}

			// HEAD request skip copy body
			if req.Method == http.MethodHead {
				return
			}

		}

		doCopyBody()
	}
}

func (s *HTTPServer) buildEndpoint() (http.HandlerFunc, error) {
	return nil, nil
}
