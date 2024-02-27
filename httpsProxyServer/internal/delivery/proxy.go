package delivery

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"time"
)

// Proxy is a forward proxy that substitutes its own certificate
// for incoming Tls connections in place of the upstream server's
// certificate.
type Proxy struct {

	// Wrap specifies a function for optionally wrapping upstream for
	// inspecting the decrypted HTTP request and response.
	Wrap func(upstream http.Handler, isSecure bool) http.Handler

	// CA specifies the root CA for generating leaf certs for each incoming
	// Tls request.
	CA *tls.Certificate

	// TlsClientConfig specifies the tls.Config to use when establishing
	// an upstream connection for proxying.
	TlsServerConfig *tls.Config

	// TLSClientConfig specifies the tls.Config to use when establishing
	// an upstream connection for proxying.
	TlsClientConfig *tls.Config

	// FlushInterval specifies the flush interval
	// to flush to the client while copying the
	// response body.
	// If zero, no periodic flushing is done.
	FlushInterval time.Duration
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {

		p.serveConnect(w, r)
		return
	}

	reverseProxy := &httputil.ReverseProxy{
		Director:      httpDirector,
		FlushInterval: p.FlushInterval,
	}
	p.Wrap(reverseProxy, false).ServeHTTP(w, r)
}

func httpDirector(r *http.Request) {
	r.URL.Host = r.Host
	r.URL.Scheme = "http"
}

func httpsDirector(r *http.Request) {
	r.URL.Host = r.Host
	r.URL.Scheme = "https"
}
