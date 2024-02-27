package delivery

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"proxyServer/mongo/domain"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Middleware struct {
	strg Storage
}

func GetMiddleware(storage Storage) Middleware {
	return Middleware{strg: storage}
}

type customRecorder struct {
	http.ResponseWriter

	response []byte
	code     int
}

func (w *customRecorder) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *customRecorder) Write(b []byte) (int, error) {
	w.response = append(w.response, b...)
	return w.ResponseWriter.Write(b)
}

func GetReqHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for name, values := range r.Header {
		if name != "Cookie" {
			headers[name] = values[0]
		}
	}
	return headers
}

func GetReqCookies(r *http.Request) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}

func GetReqGetParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	query := r.URL.Query()
	for key, values := range query {
		params[key] = values[0]
	}
	return params
}

func GetReqPostParams(requestBody []byte) map[string]string {
	form, _ := url.ParseQuery(string(requestBody))
	params := make(map[string]string)
	for key, values := range form {
		params[key] = values[0]
	}
	return params
}

func GetResHeaders(w http.ResponseWriter) map[string]string {
	headers := make(map[string]string)
	for name, values := range w.Header() {
		headers[name] = values[0]
	}
	return headers
}

func (mw *Middleware) Save(upstream http.Handler, isSecure bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.Host, r.URL.Path)
		r.Header.Set("X-From-Proxy", "yes")

		recorder := &customRecorder{ResponseWriter: w}

		reqBody, _ := io.ReadAll(r.Body)
		bodyReader := io.NopCloser(bytes.NewBuffer(reqBody))

		r.Body = bodyReader

		r.Header.Del("Accept-Encoding")
		recorder.Header().Set("Content-Encoding", "identity")
		objectID := primitive.NewObjectID()
		recorder.Header().Set("X-Transaction-Id", objectID.Hex())

		reqGetParams := GetReqGetParams(r)
		reqHeaders := GetReqHeaders(r)
		reqCookies := GetReqCookies(r)

		reqPostParams := make(map[string]string)
		if reqHeaders["Content-Type"] == "application/x-www-form-urlencoded" {
			reqPostParams = GetReqPostParams(reqBody)
		}

		var err error
		var protocol string

		if isSecure {
			protocol = "https"
		} else {
			protocol = "http"
		}

		upstream.ServeHTTP(recorder, r)

		GetResHeaders := GetResHeaders(recorder)

		var resTextBody string
		if strings.Contains(GetResHeaders["Content-Type"], "text") ||
			(strings.Contains(GetResHeaders["Content-Type"], "application") && !strings.Contains(GetResHeaders["Content-Type"], "application/octet-stream")) {
			resTextBody = string(recorder.response)
		}

		transaction := domain.HTTPTransaction{
			ID:   objectID,
			Time: time.Now(),
			Request: domain.Request{
				Host:       r.Host,
				Method:     r.Method,
				Version:    r.Proto,
				Path:       r.URL.Path,
				Headers:    reqHeaders,
				Cookies:    reqCookies,
				Protocol:   protocol,
				GetParams:  reqGetParams,
				PostParams: reqPostParams,
				RawBody:    reqBody,
			},
			Response: domain.Response{
				StatusCode:    recorder.code,
				RawBody:       recorder.response,
				TextBody:      resTextBody,
				Headers:       GetResHeaders,
				ContentLenght: len(recorder.response),
			},
		}

		err = mw.strg.Add(transaction)
		if err != nil {
			http.Error(w, "Error to add request to db", http.StatusInternalServerError)
			log.Println("error to add request to db", err)
			return
		}

	})
}
