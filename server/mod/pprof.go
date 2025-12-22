package mod

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/http/pprof"

	"github.com/omalloc/tavern/conf"
)

var uname, passw [32]byte

func HandlePProf(c *conf.ServerPProf, r *http.ServeMux) {
	uname = sha256.Sum256([]byte(c.Username))
	passw = sha256.Sum256([]byte(c.Password))

	r.HandleFunc("/debug/pprof/", basicAuth(pprof.Index))
	r.HandleFunc("/debug/pprof/cmdline", basicAuth(pprof.Cmdline))
	r.HandleFunc("/debug/pprof/profile", basicAuth(pprof.Profile))
	r.HandleFunc("/debug/pprof/symbol", basicAuth(pprof.Symbol))
	r.HandleFunc("/debug/pprof/trace", basicAuth(pprof.Trace))
}

// basicAuth is a middleware function that enforces HTTP Basic Authentication for a given handler function.
//
// e.g.
//
//		echo -n "root:password" | base64
//	 	> cm9vdDpwYXNzd29yZA==
//		curl http://api/debug/pprof/profile -H'Authorization: Basic cm9vdDpwYXNzd29yZA=='
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			// Calculate SHA-256 hashes for the provided and expected
			// usernames and passwords.
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))

			// Use the subtle.ConstantTimeCompare() function to check if
			// the provided username and password hashes equal the
			// expected username and password hashes. ConstantTimeCompare
			// will return 1 if the values are equal, or 0 otherwise.
			// Importantly, we should to do the work to evaluate both the
			// username and password before checking the return values to
			// avoid leaking information.
			usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], uname[:]) == 1
			passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], passw[:]) == 1

			// If the username and password are correct, then call
			// the next handler in the chain. Make sure to return
			// afterwards, so that none of the code below is run.
			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		// If the Authentication header is not present, is invalid, or the
		// username or password is wrong, then set a WWW-Authenticate
		// header to inform the client that we expect them to use basic
		// authentication and send a 401 Unauthorized response.
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}
