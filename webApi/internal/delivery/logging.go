package delivery

import (
	"log"
	"net/http"
)

func Log(upstream http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.Host, r.URL.Path)
		r.Header.Del("If-Modified-Since")
		r.Header.Del("If-None-Match")
		upstream.ServeHTTP(w, r)
	})
}
