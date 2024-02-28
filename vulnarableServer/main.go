package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Host, r.URL.Path, r.Method)
	params := r.URL.Query()
	name := params.Get("name")
	response := fmt.Sprintf("Hello, %s!", name)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("listen :8086")
	log.Fatal(http.ListenAndServe(":8086", nil))
}
