package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RequestPayload struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Form     map[string]string `json:"form"`
	Timeout  int               `json:"timeout"`
	IsBinary bool              `json:"is_binary"`
}

type ResponsePayload struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	IsBase64   bool              `json:"is_base64"`
	Error      string            `json:"error,omitempty"`
}

var client = &http.Client{
	Timeout: 20 * time.Second,
}

func HandleProxyHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(ResponsePayload{Error: "Method not allowed"})
		return
	}

	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ResponsePayload{Error: "Invalid JSON request: " + err.Error()})
		return
	}

	if payload.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ResponsePayload{Error: "URL cannot be empty"})
		return
	}

	method := strings.ToUpper(payload.Method)
	if method == "" {
		method = http.MethodGet
	}

	var reqBody io.Reader
	if len(payload.Form) > 0 {
		values := url.Values{}
		for k, v := range payload.Form {
			values.Set(k, v)
		}
		reqBody = strings.NewReader(values.Encode())
	} else if payload.Body != "" {
		reqBody = bytes.NewReader([]byte(payload.Body))
	}

	req, err := http.NewRequest(method, payload.URL, reqBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ResponsePayload{Error: "Failed to construct request: " + err.Error()})
		return
	}

	// Set default headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	if len(payload.Form) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	for k, v := range payload.Headers {
		req.Header.Set(k, v)
	}

	timeout := 15 * time.Second
	if payload.Timeout > 0 {
		timeout = time.Duration(payload.Timeout) * time.Millisecond
	}
	customClient := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := customClient.Do(req)
	if err != nil {
		_ = json.NewEncoder(w).Encode(ResponsePayload{
			StatusCode: 502,
			Error:      err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	respHeaders := make(map[string]string)
	for k, vv := range resp.Header {
		respHeaders[k] = strings.Join(vv, "; ")
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = json.NewEncoder(w).Encode(ResponsePayload{
			StatusCode: resp.StatusCode,
			Headers:    respHeaders,
			Error:      "Failed to read response body: " + err.Error(),
		})
		return
	}

	isBinary := payload.IsBinary || isBinaryContentType(resp.Header.Get("Content-Type"))
	bodyStr := ""
	if isBinary {
		bodyStr = base64.StdEncoding.EncodeToString(respData)
	} else {
		bodyStr = string(respData)
	}

	_ = json.NewEncoder(w).Encode(ResponsePayload{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       bodyStr,
		IsBase64:   isBinary,
	})
}

func isBinaryContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "audio/") ||
		strings.HasPrefix(ct, "image/") ||
		strings.HasPrefix(ct, "video/") ||
		strings.Contains(ct, "octet-stream") ||
		strings.Contains(ct, "zip") ||
		strings.Contains(ct, "gzip")
}
