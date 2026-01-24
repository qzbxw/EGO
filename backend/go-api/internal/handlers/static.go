package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// StaticHandler serves static assets like images, logos, and icons.
type StaticHandler struct {
	staticDir string
}

// NewStaticHandler creates a new handler for serving static files.
// It takes the root directory of the static files as an argument.
func NewStaticHandler(staticDir string) *StaticHandler {
	// Ensure the static directory path is clean and absolute for security.
	absPath, err := filepath.Abs(staticDir)
	if err != nil {
		// In a real application, this should probably cause a panic
		// as it's a configuration error at startup.
		panic("Failed to resolve absolute path for static directory: " + err.Error())
	}
	return &StaticHandler{
		staticDir: absPath,
	}
}

// RegisterRoutes registers the static file serving route with the Chi router.
func (h *StaticHandler) RegisterRoutes(r chi.Router) {
	// The path '/static/' is the URL prefix.
	// http.StripPrefix removes this prefix before the file server looks for the file.
	// For example, a request to '/static/images/logo.png' will make the file server
	// look for './static/images/logo.png' in the filesystem.
	fileServer := http.FileServer(http.Dir(h.staticDir))

	r.Route("/static", func(r chi.Router) {
		// Apply a caching middleware to all static assets.
		r.Use(h.cachingMiddleware)
		r.Handle("/*", http.StripPrefix("/static", fileServer))
	})
}

// cachingMiddleware sets the Cache-Control header for static assets.
func (h *StaticHandler) cachingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set a long cache duration for static files (e.g., 24 hours).
		w.Header().Set("Cache-Control", "public, max-age=86400")
		next.ServeHTTP(w, r)
	})
}
