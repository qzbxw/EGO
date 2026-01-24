package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// PythonProxyHandler is a generic reverse proxy that forwards requests
// to the Python backend service while preserving the path (minus the prefix).
type PythonProxyHandler struct {
	proxy *httputil.ReverseProxy
}

// NewPythonProxyHandler creates a new reverse proxy targeting the specified Python backend URL.
func NewPythonProxyHandler(targetURL string) (*PythonProxyHandler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL for python proxy: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom Director to rewrite the request path correctly.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// The path is already modified by chi's StripPrefix if we use it,
		// but we want to be sure it's correct for the target.
		req.Host = target.Host
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Python Proxy error: %v", err)
		RespondWithError(w, http.StatusBadGateway, "The background processing service is currently unavailable.")
	}

	return &PythonProxyHandler{proxy: proxy}, nil
}

// HandleProxy forwards the incoming HTTP request to the Python backend.
func (h *PythonProxyHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}

// TitleProxyHandler is a specialized reverse proxy that forwards requests
// for title generation to the Python backend service.
type TitleProxyHandler struct {
	proxy *httputil.ReverseProxy
}

// NewTitleProxyHandler creates a new reverse proxy targeting the specified Python backend URL.
// It returns an error if the target URL is invalid.
func NewTitleProxyHandler(targetURL string) (*TitleProxyHandler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL for proxy: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom Director to rewrite the request path.
	// This ensures that any request to this handler is always sent
	// to the correct '/generate_title' endpoint on the Python service.
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = "/generate_title"
		req.Host = target.Host // Set the Host header to the target's host.
	}

	// It's good practice to have an error handler for the proxy.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		RespondWithError(w, http.StatusBadGateway, "The content generation service is currently unavailable.")
	}

	return &TitleProxyHandler{proxy: proxy}, nil
}

// HandleProxy forwards the incoming HTTP request to the Python backend.
func (h *TitleProxyHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}
