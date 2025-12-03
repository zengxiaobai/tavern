package mod

import (
	"net/http"

	"github.com/omalloc/tavern/metrics"
	xhttp "github.com/omalloc/tavern/pkg/x/http"
)

func fillRequest(req *http.Request) {
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
		if req.TLS != nil {
			req.URL.Scheme = "https"
		}
	}
	if req.URL.Host == "" {
		req.URL.Host = req.Host
	}
}

func wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		fillRequest(req)

		req, metric := metrics.WithRequestMetric(req)

		rw := xhttp.NewResponseRecorder(w)
		defer func() {
			metric.SentResp = rw.SentBytes()
		}()

		next(rw, req)
	}
}
