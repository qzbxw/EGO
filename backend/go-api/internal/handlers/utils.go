package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RespondWithError writes a standard JSON error response with a given status code.
// For server-side errors (5xx), it returns a generic message to avoid leaking
// internal implementation details to the client.
func RespondWithError(w http.ResponseWriter, code int, message string) {
	// For 500 Internal Server Error, log the specific message for debugging but
	// send a generic message to the client for security.
	// Other 5xx codes like 503 Service Unavailable should keep their messages.
	if code == http.StatusInternalServerError {
		log.Printf("Responding with server error (%d): %s", code, message)
		message = "An internal server error occurred. Please try again later."
	}
	RespondWithJSON(w, code, map[string]string{"error": message})
}

// RespondWithJSON marshals a payload to JSON, sets the appropriate headers,
// and writes the response with a given status code.
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		// If marshaling fails, it's a server-side programming error.
		log.Printf("!!! CRITICAL: Failed to marshal JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to serialize response"}`)) // Fallback response
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(response)
}

// getClientIP extracts the client's real IP address from the request headers.
// It checks common proxy headers like 'X-Forwarded-For' and 'X-Real-IP' first,
// falling back to the request's RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check the 'X-Forwarded-For' header, which can contain a comma-separated list of IPs.
	// The first IP in the list is the original client's IP.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check the 'X-Real-IP' header, often set by reverse proxies like Nginx.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to the direct network address from the request.
	// This might be the IP of a load balancer or the client itself.
	// We trim the port number if it exists.
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// parseIDFromURL extracts a numeric ID from a URL parameter using Chi.
// It returns an error if the parameter is not a valid 64-bit integer.
func parseIDFromURL(r *http.Request, key string) (int64, error) {
	idStr := chi.URLParam(r, key)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// IsBrowserRequest checks if an HTTP request likely came from a web browser.
func IsBrowserRequest(r *http.Request) bool {
	acceptHeader := r.Header.Get("Accept")
	return strings.Contains(acceptHeader, "text/html")
}
