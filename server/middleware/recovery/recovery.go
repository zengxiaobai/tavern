package recovery

import (
	"net/http"

	configv1 "github.com/omalloc/tavern/api/defined/v1/middleware"
	"github.com/omalloc/tavern/contrib/log"
	"github.com/omalloc/tavern/pkg/x/runtime"
	"github.com/omalloc/tavern/server/middleware"
)

func init() {
	middleware.Register("recovery", Middleware)
}

type middlewareOption struct{}

func Middleware(c *configv1.Middleware) (middleware.Middleware, func(), error) {
	var opts middlewareOption
	if err := c.Unmarshal(&opts); err != nil {
		return nil, nil, err
	}

	return func(origin http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
			defer func() {
				if r := recover(); r != nil {
					// Here you can log the panic or handle it as needed
					log.Context(req.Context()).Errorf("middleware recovery: %s \n%s", r, runtime.PrintStackTrace(4))
				}
			}()

			return origin.RoundTrip(req)
		})
	}, middleware.EmptyCleanup, nil
}
