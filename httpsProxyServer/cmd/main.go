package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"path"
	"proxyServer/httpsProxyServer/internal/delivery"
	"proxyServer/mongo/mongoclient"
	"proxyServer/mongo/storage"

	certLib "proxyServer/httpsProxyServer/pkg/cert"
)

var (
	hostname, _ = os.Hostname()

	dir      = path.Join(os.Getenv("HOME"), ".mitm")
	keyFile  = path.Join(dir, "ca-key.pem")
	certFile = path.Join(dir, "ca-cert.pem")
)

const URI = "mongodb://root:root@localhost:27017"

func main() {
	log.SetPrefix("[PROXY] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	client, closeConn, err := mongoclient.CreateMongoClient(URI)
	if err != nil {
		log.Fatal(err)
	}
	defer closeConn()

	strg, err := storage.CreateStorage(client)
	if err != nil {
		log.Fatal(err)
	}
	middleware := delivery.GetMiddleware(&strg)

	ca, err := loadCA()
	if err != nil {
		log.Fatal(err)
	}

	proxyHandler := &delivery.Proxy{
		CA: &ca,
		TlsServerConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		Wrap: middleware.Save,
	}

	log.Println("proxy :8080")
	log.Fatal(http.ListenAndServe(":8080", proxyHandler))
}

func loadCA() (cert tls.Certificate, err error) {
	cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	if os.IsNotExist(err) {
		cert, err = genCA()
	}
	if err == nil {
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	}
	return
}

func genCA() (cert tls.Certificate, err error) {
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	certPEM, keyPEM, err := certLib.GenCA(hostname)
	if err != nil {
		return
	}
	cert, _ = tls.X509KeyPair(certPEM, keyPEM)
	err = os.WriteFile(certFile, certPEM, 0400)
	if err == nil {
		err = os.WriteFile(keyFile, keyPEM, 0400)
	}
	return cert, err
}
