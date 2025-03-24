package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v10"
)

var (
	// view & rotate api key ?
	forbidAPIs = map[string]struct{}{
		"DELETE /api/accounts":                   {}, // delete account
		"PUT /api/accounts/profile":              {}, // change name
		"POST /api/organizations/{org_id}/leave": {}, // leave org
	}

	checkIAPFields = map[string]string{
		"/prelogin":               "email",    // prelogin to get parameters
		"/identity/connect/token": "username", // real login
		// TODO: change pwd
	}
)

type appArgs struct {
	BindAddr    string  `env:"BIND_ADDR" envDefault:":8080"`
	UpStreamURL url.URL `env:"UP_STREAM_URL"`
}

func main() {
	args := appArgs{}
	if err := env.Parse(&args); err != nil {
		log.Fatal("err:", err)
	}

	// TODO: basic args validation

	handler := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			// following operation is forbidden
			if _, ok := forbidAPIs[fmt.Sprintf("%s %s", r.Method, r.URL.Path)]; ok {
				w.WriteHeader(http.StatusNotAcceptable)
				return
			}

			if field, ok := checkIAPFields[r.URL.Path]; ok {
				if (r.Method == http.MethodPost || r.Method == http.MethodPut) && !checkIAP(w, r, field) {
					return
				}
			}

			if r.URL.Path == "/" {
				// for auto fill email in login form
				// add r=1 to prevent redirect loops
				if r.URL.Query().Get("r") != "1" {
					email := getIAPHeader(w, r)
					if email == "" {
						return
					}
					http.Redirect(w, r, "/?r=1#/login?email="+email, http.StatusTemporaryRedirect)
					return
				}
			}

			// need more access logs ?
			r.Host = args.UpStreamURL.Host
			p.ServeHTTP(w, r)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(&args.UpStreamURL)
	http.HandleFunc("/", handler(proxy))

	err := http.ListenAndServe(args.BindAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func checkIAP(w http.ResponseWriter, r *http.Request, field string) bool {
	email := getIAPHeader(w, r)
	if email == "" {
		return false
	}
	values, err := parseRequestToMap(r)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "BadRequest: Fail to parse body", http.StatusBadRequest)
		return false
	}
	if v, ok := values[field]; ok && v != email {
		http.Error(w, "Unauthorized: illegail operation", http.StatusUnauthorized)
		return false
	}
	return true
}

func getIAPHeader(w http.ResponseWriter, r *http.Request) string {
	// Get the IAP user email header
	emailHeader := r.Header.Get("X-Goog-Authenticated-User-Email")
	if emailHeader == "" {
		http.Error(w, "Unauthorized: IAP header missing", http.StatusUnauthorized)
		return ""
	}

	// Extract the email part: accounts.google.com:email@example.com → email@example.com
	parts := strings.SplitN(emailHeader, ":", 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid email header format", http.StatusBadRequest)
		return ""
	}
	return parts[1]
}

func parseRequestToMap(req *http.Request) (map[string]interface{}, error) {
	ct := req.Header.Get("Content-Type")
	result := make(map[string]interface{})

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	// reset body for reverse proxy
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))

	switch {
	case strings.HasPrefix(ct, "application/json"):
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	case strings.HasPrefix(ct, "application/x-www-form-urlencoded"):
		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("invalid form: %w", err)
		}
		for key, val := range values {
			if len(val) == 1 {
				result[key] = val[0]
			} else {
				result[key] = val
			}
		}
	default:
		return nil, fmt.Errorf("unsupported content-type: %s", ct)
	}

	return result, nil
}
