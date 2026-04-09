package webserver

import (
	"net/http"
)

// BasicAuthMiddleware returns an HTTP middleware that requires HTTP Basic Auth
// with the given password. The username is ignored — only the password matters.
//
// On missing or incorrect credentials, the middleware responds with:
//   - HTTP 401 Unauthorized
//   - WWW-Authenticate: Basic realm="AgentHub"
//
// This is used in local mode to protect the web server from unauthenticated
// access on the local network.
func BasicAuthMiddleware(password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, pass, ok := r.BasicAuth()
			if !ok || pass != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="AgentHub"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// basicAuthMiddleware is the unexported alias used internally by startLocal.
// It delegates to BasicAuthMiddleware so tests can verify via the exported form.
func basicAuthMiddleware(password string) func(http.Handler) http.Handler {
	return BasicAuthMiddleware(password)
}
