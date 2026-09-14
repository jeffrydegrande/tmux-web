// Package web holds the embedded HTML templates and static assets. The web
// server links this package. A Go embed pattern can not reach a parent
// directory, so the embed lives here next to the files, not in cmd/web.
package web

import "embed"

// TemplateFS holds the HTML templates under templates/.
//
//go:embed templates/*.html
var TemplateFS embed.FS

// StaticFS holds the CSS and the embedded HTMX script under static/.
//
//go:embed static/*
var StaticFS embed.FS
