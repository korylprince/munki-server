package main

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuthHandler is a HTTP Basic Auth handler
func BasicAuthHandler(username, password string, next http.Handler) http.Handler {
	user := []byte(username)
	userLen := int32(len(user))
	pass := []byte(password)
	passLen := int32(len(pass))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeEq(userLen, int32(len([]byte(u)))) != 1 ||
			subtle.ConstantTimeCompare(user, []byte(u)) != 1 ||
			subtle.ConstantTimeEq(passLen, int32(len([]byte(p)))) != 1 ||
			subtle.ConstantTimeCompare(pass, []byte(p)) != 1 {
			w.Header().Set("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
