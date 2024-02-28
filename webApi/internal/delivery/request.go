package delivery

import (
	"bytes"
	"encoding/json"
	"io"
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
			Headers:       tr.Request.Headers,
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

	err = resRepeat.Write(w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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

	query := u.Query()
	for key, value := range transaction.Request.GetParams {
		query.Add(key, value)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequest(transaction.Request.Method,
		transaction.Request.Protocol+"://"+u.String(),
		bytes.NewBuffer(transaction.Request.RawBody))
	if err != nil {
		return nil, err
	}

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

	return resp, nil
}

func (h *Handler) ScanByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	id := mux.Vars(r)["id"]
	transaction, err := h.strg.GetByID(id)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error to get transaction by id"))
		return
	}

	vulnGetParams := []string{}
	vulnPostParams := []string{}

	transactionsIDs := []string{}

	for key, value := range transaction.Request.GetParams {
		transaction.Request.GetParams[key] = `vulnerable'"><img src onerror=alert()>`
		resRepeat, err := RepeatRequest(transaction)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error to repeat request"))
			return
		}

		body, err := io.ReadAll(resRepeat.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error to read body from repeated request"))
			return
		}

		transactionsIDs = append(transactionsIDs, resRepeat.Header[resHeaderTransactionID][0])
		if bytes.Contains(body, []byte(attackVector)) {
			vulnGetParams = append(vulnGetParams, key)
		}
		transaction.Request.GetParams[key] = value
	}

	for key, value := range transaction.Request.PostParams {
		transaction.Request.PostParams[key] = `vulnerable'"><img src onerror=alert()>`
		resRepeat, err := RepeatRequest(transaction)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error to repeat request"))
			return
		}
		body, err := io.ReadAll(resRepeat.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error to read body from repeated request"))
			return
		}
		transactionsIDs = append(transactionsIDs, resRepeat.Header[resHeaderTransactionID][0])
		if bytes.Contains(body, []byte(attackVector)) {
			vulnPostParams = append(vulnGetParams, key)
		}

		transaction.Request.PostParams[key] = value
	}

	isVuln := false
	if len(vulnGetParams)+len(vulnPostParams) != 0 {
		isVuln = true
	}

	res, err := json.Marshal(map[string]interface{}{
		"body": map[string]interface{}{
			"request_id":             id,
			"scan_requests":          transactionsIDs,
			"is_vulnerable":          isVuln,
			"vulnerable_post_params": vulnPostParams,
			"vulnerable_get_params":  vulnGetParams,
		},
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(res)
}
