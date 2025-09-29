// Package templates handles parsing and rendering of HTML templates.
package templates

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed *.html
var templateFiles embed.FS

var (
	// parsedTemplates holds all parsed templates, ready for execution.
	// It's initialized once at startup.
	parsedTemplates *template.Template
)

func init() {
	var err error
	parsedTemplates, err = template.ParseFS(templateFiles, "*.html")
	if err != nil {
		// Panic at startup if templates are missing or invalid.
		panic("Failed to parse HTML templates: " + err.Error())
	}
}

// PageData contains all the dynamic data needed to render a status page.
type PageData struct {
	Title             string
	StatusText        string
	Subtitle          string
	Message           string
	IsMaintenanceMode bool
}

// RenderStatusPage renders the HTML status page (maintenance.html) with the appropriate data.
func RenderStatusPage(w http.ResponseWriter, isMaintenanceMode bool, customMessage string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var data PageData
	if isMaintenanceMode {
		w.WriteHeader(http.StatusServiceUnavailable)
		data = PageData{
			Title:             "Service Unavailable",
			StatusText:        "UNAVAILABLE",
			Subtitle:          "We are currently performing maintenance.",
			Message:           customMessage,
			IsMaintenanceMode: true,
		}
	} else {
		w.WriteHeader(http.StatusOK)
		data = PageData{
			Title:             "System Status: OK",
			StatusText:        "AVAILABLE",
			IsMaintenanceMode: false,
		}
	}

	// Execute the named template from our parsed set.
	return parsedTemplates.ExecuteTemplate(w, "maintenance.html", data)
}