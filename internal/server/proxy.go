package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type legacyProxy struct {
	proxy *httputil.ReverseProxy
}

func newLegacyProxy(rawURL string) (*legacyProxy, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, nil
	}

	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	return &legacyProxy{proxy: proxy}, nil
}

func (p *legacyProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
