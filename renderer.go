package forge

import "io"

// Renderer is the interface for HTML template engines.
// The default implementation uses html/template.
// Replace it with your own via forge.SetRenderer().
type Renderer interface {
	Render(w io.Writer, template string, data any) error
}

// globalRenderer is the active renderer. Nil until one is configured.
var globalRenderer Renderer

// SetRenderer registers a custom renderer (e.g. templ, jet, pongo2).
func SetRenderer(r Renderer) {
	globalRenderer = r
}
