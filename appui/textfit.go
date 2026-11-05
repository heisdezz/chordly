package appui

import "github.com/gogpu/ui/widget"

// fitText trims text to maxWidth at the given font settings, appending an
// ellipsis when it does not fit. Canvases in GoGPU do not wrap or clip text,
// so painters must fit strings themselves.
func fitText(canvas widget.Canvas, text string, fontSize float32, bold bool, maxWidth float32) string {
	if text == "" || maxWidth <= 0 {
		return ""
	}
	if canvas.MeasureText(text, fontSize, bold) <= maxWidth {
		return text
	}

	const ellipsis = "…"
	ellipsisWidth := canvas.MeasureText(ellipsis, fontSize, bold)
	if ellipsisWidth > maxWidth {
		return ""
	}

	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		width := canvas.MeasureText(string(runes[:mid]), fontSize, bold) + ellipsisWidth
		if width <= maxWidth {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return string(runes[:lo]) + ellipsis
}
