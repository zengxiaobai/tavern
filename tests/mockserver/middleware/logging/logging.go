package logging

import (
	"log"
	"net/http"
)

func Logging(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.String())

		next.ServeHTTP(w, r)
	}
}
