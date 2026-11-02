package main

import "github.com/egoist/mygo/ui"

// Small stroke icons (24×24, drawn for this app).
func svgIcon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	icBluetooth = svgIcon(`<path d="m7 7 10 10-5 5V2l5 5L7 17"/>`)
	icCheck     = svgIcon(`<path d="M20 6 9 17l-5-5"/>`)
	icCheckDisc = svgIcon(`<circle cx="12" cy="12" r="10"/><path d="m8 12 3 3 5-6"/>`)
	icX         = svgIcon(`<path d="M18 6 6 18M6 6l12 12"/>`)
	icAlert     = svgIcon(`<path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/><path d="M12 9v4M12 17h.01"/>`)
	icRefresh   = svgIcon(`<path d="M21 12a9 9 0 1 1-2.6-6.4L21 8"/><path d="M21 3v5h-5"/>`)
	icChip      = svgIcon(`<rect x="5" y="5" width="14" height="14" rx="2"/><path d="M9 9h6v6H9zM9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3"/>`)
	icBook      = svgIcon(`<path d="M2 4h6a4 4 0 0 1 4 4v13a3 3 0 0 0-3-3H2zM22 4h-6a4 4 0 0 0-4 4v13a3 3 0 0 1 3-3h7z"/>`)
	icDownload  = svgIcon(`<path d="M12 3v12M7 10l5 5 5-5M5 21h14"/>`)
)
