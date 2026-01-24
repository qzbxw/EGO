// Package templates handles parsing and rendering of HTML templates.
package templates

import (
	"html/template"
)

var (
	// parsedTemplates holds all parsed templates, ready for execution.
	// It's initialized once at startup.
	parsedTemplates *template.Template
)

func init() {
	// If we decide to add HTML templates back in the future, 
	// we can re-enable template parsing here.
	parsedTemplates = template.New("empty")
}
