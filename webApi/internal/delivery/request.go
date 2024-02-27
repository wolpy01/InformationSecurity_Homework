package delivery

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"proxyServer/mongo/domain"

	"github.com/gorilla/mux"
)

type Handler struct {
	strg Storage
}

func GetHandler(strg Storage) Handler {
	return Handler{strg: strg}
}

func (h *Handler) Requests(w http.ResponseWriter, r *http.Request) {
	transactions, err := h.strg.GetAll()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error to get all requests"))
	}

	reqs := []TransactionDTO{}
	for _, tr := range transactions {
		req := TransactionDTO{
			ID:            tr.ID.(string),
			Host:          tr.Request.Host,
			Method:        tr.Request.Method,
			Path:          tr.Request.Path,
			StatusCode:    tr.Response.StatusCode,
			ContentLenght: tr.Response.ContentLenght,
		}
		reqs = append(reqs, req)
	}
	response, err := json.Marshal(reqs)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(response)

}

func (h *Handler) RequestByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	id := mux.Vars(r)["id"]
	transaction, err := h.strg.GetByID(id)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error to get transaction by id"))
	}

	response, err := json.Marshal(transaction)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (h *Handler) RepeatByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	id := mux.Vars(r)["id"]
	transaction, err := h.strg.GetByID(id)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error to get transaction by id"))
		return
	}
	
	resRepeat, err := RepeatRequest(transaction)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error to repeat request"))
		return

	}
	res, err := json.Marshal(map[string]interface{}{
		"body": map[string]interface{}{
			"current_request_id":  resRepeat.Header[resHeaderTransactionID][0],
			"repeated_request_id": id,
			"status_code":         resRepeat.Status,
			"content_length":      resRepeat.ContentLength,
		},
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Write(res)
	w.WriteHeader(http.StatusOK)
}

func RepeatRequest(transaction domain.HTTPTransaction) (*http.Response, error) {
	proxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}

	u, err := url.Parse(transaction.Request.Host + transaction.Request.Path)
	if err != nil {
		return nil, err
	}
	log.Println(u)
	query := u.Query()
	for key, value := range transaction.Request.GetParams {
		query.Add(key, value)
	}
	u.RawQuery = query.Encode()
	log.Println(query)

	req, err := http.NewRequest(transaction.Request.Method,
		transaction.Request.Protocol+"://"+u.String(),
		bytes.NewBuffer(transaction.Response.RawBody))
	if err != nil {
		return nil, err
	}
	log.Println("Made request")

	for key, value := range transaction.Request.Headers {
		req.Header.Set(key, value)
	}

	for key, value := range transaction.Request.Cookies {
		req.AddCookie(&http.Cookie{Name: key, Value: value})
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	log.Println("Got response")

	return resp, nil
}
