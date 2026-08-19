package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewUIProxy(rawURL string) (http.Handler, error) {
	target, err := url.ParseRequestURI(rawURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("parse PA_UI_DEV_PROXY %q: %w", rawURL, err)
	}
	return httputil.NewSingleHostReverseProxy(target), nil
}
