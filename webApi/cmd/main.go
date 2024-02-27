package main

import (
	"log"
	"net/http"
	"proxyServer/mongo/mongoclient"
	"proxyServer/mongo/storage"
	"proxyServer/webApi/internal/delivery"

	"github.com/gorilla/mux"
)

const URI = "mongodb://root:root@localhost:27017"

func main() {
	log.SetPrefix("[WEB-API] ")
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

	handler := delivery.GetHandler(&strg)

	r := mux.NewRouter()

	r.Use(delivery.Log)

	r.HandleFunc("/requests", handler.Requests)
	r.HandleFunc("/requests/{id}", handler.RequestByID)
	r.HandleFunc("/repeat/{id}", handler.RepeatByID)

	log.Println("web-api :8000")
	log.Fatal(http.ListenAndServe(":8000", r))
}
