package delivery

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"proxyServer/httpsProxyServer/pkg/cert"
)

// Proxy is a forward proxy that substitutes its own certificate
// for incoming TLS connections in place of the upstream server's
// certificate.
func (p *Proxy) serveConnect(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("Accept-Encoding")
	var (
		err        error
		targetConn *tls.Conn
		targetName = dnsName(r.Host)
	)

	if targetName == "" {
		log.Println("cannot determine cert name for " + r.Host)
		http.Error(w, "no upstream", 503)
		return
	}

	tmpCert, err := cert.GenCert(p.CA, []string{targetName})
	if err != nil {
		log.Println("cert targetName", err)
		http.Error(w, "no upstream", 503)
		return
	}

	serverProxyConfig := new(tls.Config)
	if p.TlsServerConfig != nil {
		*serverProxyConfig = *p.TlsServerConfig
	}

	serverProxyConfig.Certificates = []tls.Certificate{*tmpCert}
	serverProxyConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		clientProxyConfig := new(tls.Config)
		if p.TlsClientConfig != nil {
			*clientProxyConfig = *p.TlsClientConfig
		}
		clientProxyConfig.ServerName = hello.ServerName
		targetConn, err = tls.Dial("tcp", r.Host, clientProxyConfig)
		if err != nil {
			log.Println("dial", r.Host, err)
			return nil, err
		}
		return cert.GenCert(p.CA, []string{hello.ServerName})
	}

	clientConn, err := connectClient(w, serverProxyConfig)
	if err != nil {
		log.Println("handshake", r.Host, err)
		return
	}
	defer clientConn.Close()
	if targetConn == nil {
		log.Println("could not determine cert name for " + r.Host)
		return
	}
	defer targetConn.Close()

	dialer := &oneShotDialer{targetConn: targetConn}
	reverseProxy := &httputil.ReverseProxy{
		Director:      httpsDirector,
		Transport:     &http.Transport{DialTLS: dialer.Dial},
		FlushInterval: p.FlushInterval,
	}

	ch := make(chan int)
	clientConnFastClose := &onCloseConn{clientConn, func() { ch <- 0 }}
	http.Serve(&oneShotListener{clientConnFastClose}, p.Wrap(reverseProxy, true))
	<-ch
}

// handshake hijacks w's underlying net.Conn, responds to the CONNECT request
// and manually performs the TLS handshake. It returns the net.Conn or and
// error if any.
func connectClient(w http.ResponseWriter, config *tls.Config) (net.Conn, error) {
	rawClientConn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "no upstream", 503)
		return nil, err
	}
	if _, err = rawClientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		rawClientConn.Close()
		return nil, err
	}

	clientConn := tls.Server(rawClientConn, config)
	err = clientConn.Handshake()

	if err != nil {
		clientConn.Close()
		rawClientConn.Close()
		return nil, err
	}
	return clientConn, nil
}

// dnsName returns the DNS name in addr, if any.
func dnsName(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return host
}

func (d *oneShotDialer) Dial(network, addr string) (net.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.targetConn == nil {
		return nil, errors.New("closed")
	}
	targetConn := d.targetConn
	d.targetConn = nil
	return targetConn, nil
}

// A oneShotListener implements net.Listener whos Accept only returns a
// net.Conn as specified by c followed by an error for each subsequent Accept.
func (l *oneShotListener) Accept() (net.Conn, error) {
	if l.clientConn == nil {
		return nil, errors.New("closed")
	}
	clientConn := l.clientConn
	l.clientConn = nil
	return clientConn, nil
}

func (l *oneShotListener) Close() error {
	return nil
}

func (l *oneShotListener) Addr() net.Addr {
	return l.clientConn.LocalAddr()
}

func (c *onCloseConn) Close() error {
	if c.f != nil {
		c.f()
		c.f = nil
	}
	return c.Conn.Close()
}
