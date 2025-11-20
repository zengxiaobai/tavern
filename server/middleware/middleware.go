package middleware

import (
	"net/http"

	configv1 "github.com/omalloc/tavern/api/defined/v1/middleware"
)

// Factory is a middleware factory.
type Factory func(*configv1.Middleware) (middleware Middleware, cleanup func(), err error)

// Middleware is handler middleware.
type Middleware func(http.RoundTripper) http.RoundTripper

// RoundTripperFunc is an adapter to allow the use of
// ordinary functions as HTTP RoundTripper.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(w, r).
func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Chain returns a Middleware that specifies the chained handler for endpoint.
func Chain(m ...Middleware) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}

var EmptyMiddleware = func(tripper http.RoundTripper) http.RoundTripper { return tripper }
var EmptyCleanup = func() {}
