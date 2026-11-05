package appui

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// IconType identifies vector-drawn UI icons.
type IconType int

const (
	IconNone IconType = iota
	IconHome
	IconExplore
	IconLibrary
	IconSettings
	IconSearch
	IconBell
	IconExpand
	IconPlay
	IconPause
	IconPrev
	IconNext
	IconShuffle
	IconRepeat
	IconVolume
	IconQueue
	IconHeart
	IconMore
	IconWaveform
	IconLyrics
	IconInfo
	IconCheck
	IconAlert
)

// DrawVectorIcon renders a crisp 2D vector icon centered in the given bounds.
func DrawVectorIcon(canvas widget.Canvas, icon IconType, bounds geometry.Rect, color widget.Color, size float32) {
	if bounds.IsEmpty() || size <= 0 {
		return
	}

	cx := bounds.Min.X + bounds.Width()/2
	cy := bounds.Min.Y + bounds.Height()/2
	hs := size / 2

	switch icon {
	case IconHome:
		// Roof
		canvas.DrawLine(geometry.Pt(cx-hs, cy+1), geometry.Pt(cx, cy-hs), color, 2)
		canvas.DrawLine(geometry.Pt(cx, cy-hs), geometry.Pt(cx+hs, cy+1), color, 2)
		// Walls
		canvas.DrawLine(geometry.Pt(cx-hs+2.5, cy), geometry.Pt(cx-hs+2.5, cy+hs), color, 2)
		canvas.DrawLine(geometry.Pt(cx+hs-2.5, cy), geometry.Pt(cx+hs-2.5, cy+hs), color, 2)
		// Floor
		canvas.DrawLine(geometry.Pt(cx-hs+2.5, cy+hs), geometry.Pt(cx+hs-2.5, cy+hs), color, 2)
		// Door
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx-2.5, cy+hs-5),
			Max: geometry.Pt(cx+2.5, cy+hs),
		}, color, 1)

	case IconExplore: // Compass
		r := hs - 1
		canvas.StrokeCircle(geometry.Pt(cx, cy), r, color, 1.8)
		// Needle
		canvas.DrawLine(geometry.Pt(cx-r*0.45, cy+r*0.45), geometry.Pt(cx+r*0.45, cy-r*0.45), color, 2.2)
		canvas.DrawCircle(geometry.Pt(cx, cy), 2, color)

	case IconLibrary: // Equalizer bars
		barWidth := float32(3)
		// Bar 1
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx-7, cy-hs*0.4),
			Max: geometry.Pt(cx-7+barWidth, cy+hs*0.8),
		}, color, 1.5)
		// Bar 2 (tallest)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx-1.5, cy-hs*0.85),
			Max: geometry.Pt(cx-1.5+barWidth, cy+hs*0.8),
		}, color, 1.5)
		// Bar 3
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx+4, cy-hs*0.2),
			Max: geometry.Pt(cx+4+barWidth, cy+hs*0.8),
		}, color, 1.5)

	case IconSettings: // Gear
		r := hs - 2
		canvas.StrokeCircle(geometry.Pt(cx, cy), r, color, 2)
		canvas.DrawCircle(geometry.Pt(cx, cy), r*0.4, color)
		// 4 teeth notches
		canvas.DrawLine(geometry.Pt(cx, cy-r-2), geometry.Pt(cx, cy-r+1), color, 2.5)
		canvas.DrawLine(geometry.Pt(cx, cy+r-1), geometry.Pt(cx, cy+r+2), color, 2.5)
		canvas.DrawLine(geometry.Pt(cx-r-2, cy), geometry.Pt(cx-r+1, cy), color, 2.5)
		canvas.DrawLine(geometry.Pt(cx+r-1, cy), geometry.Pt(cx+r+2, cy), color, 2.5)

	case IconSearch:
		r := hs * 0.55
		scx := cx - 2
		scy := cy - 2
		canvas.StrokeCircle(geometry.Pt(scx, scy), r, color, 2)
		// Handle
		canvas.DrawLine(geometry.Pt(scx+r*0.7, scy+r*0.7), geometry.Pt(cx+hs-1, cy+hs-1), color, 2.2)

	case IconBell:
		// Dome
		bellRect := geometry.Rect{
			Min: geometry.Pt(cx-hs*0.6, cy-hs*0.6),
			Max: geometry.Pt(cx+hs*0.6, cy+hs*0.4),
		}
		canvas.StrokeRoundRect(bellRect, color, hs*0.5, 1.8)
		// Lip
		canvas.DrawLine(geometry.Pt(cx-hs*0.8, cy+hs*0.4), geometry.Pt(cx+hs*0.8, cy+hs*0.4), color, 2)
		// Clapper
		canvas.DrawCircle(geometry.Pt(cx, cy+hs*0.7), 2, color)

	case IconExpand: // 4 corner brackets
		s := hs - 1
		arm := s * 0.45
		// Top-Left
		canvas.DrawLine(geometry.Pt(cx-s, cy-s), geometry.Pt(cx-s+arm, cy-s), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx-s, cy-s), geometry.Pt(cx-s, cy-s+arm), color, 1.8)
		// Top-Right
		canvas.DrawLine(geometry.Pt(cx+s, cy-s), geometry.Pt(cx+s-arm, cy-s), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx+s, cy-s), geometry.Pt(cx+s, cy-s+arm), color, 1.8)
		// Bottom-Left
		canvas.DrawLine(geometry.Pt(cx-s, cy+s), geometry.Pt(cx-s+arm, cy+s), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx-s, cy+s), geometry.Pt(cx-s, cy+s-arm), color, 1.8)
		// Bottom-Right
		canvas.DrawLine(geometry.Pt(cx+s, cy+s), geometry.Pt(cx+s-arm, cy+s), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx+s, cy+s), geometry.Pt(cx+s, cy+s-arm), color, 1.8)

	case IconPlay:
		// Right-facing triangle
		x0 := cx - hs*0.5
		x1 := cx + hs*0.65
		yTop := cy - hs*0.65
		yBot := cy + hs*0.65
		for y := yTop; y <= yBot; y += 1 {
			progress := (y - yTop) / (yBot - yTop)
			var w float32
			if progress <= 0.5 {
				w = progress * 2 * (x1 - x0)
			} else {
				w = (1 - progress) * 2 * (x1 - x0)
			}
			canvas.DrawLine(geometry.Pt(x0, y), geometry.Pt(x0+w, y), color, 1.5)
		}

	case IconPause:
		bw := float32(3.5)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx-bw-2, cy-hs*0.65),
			Max: geometry.Pt(cx-2, cy+hs*0.65),
		}, color, 1.5)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx+2, cy-hs*0.65),
			Max: geometry.Pt(cx+bw+2, cy+hs*0.65),
		}, color, 1.5)

	case IconPrev:
		// Bar
		canvas.DrawLine(geometry.Pt(cx-hs*0.7, cy-hs*0.6), geometry.Pt(cx-hs*0.7, cy+hs*0.6), color, 2)
		// Left Arrow
		canvas.DrawLine(geometry.Pt(cx+hs*0.5, cy-hs*0.6), geometry.Pt(cx-hs*0.4, cy), color, 2)
		canvas.DrawLine(geometry.Pt(cx-hs*0.4, cy), geometry.Pt(cx+hs*0.5, cy+hs*0.6), color, 2)

	case IconNext:
		// Right Arrow
		canvas.DrawLine(geometry.Pt(cx-hs*0.5, cy-hs*0.6), geometry.Pt(cx+hs*0.4, cy), color, 2)
		canvas.DrawLine(geometry.Pt(cx+hs*0.4, cy), geometry.Pt(cx-hs*0.5, cy+hs*0.6), color, 2)
		// Bar
		canvas.DrawLine(geometry.Pt(cx+hs*0.7, cy-hs*0.6), geometry.Pt(cx+hs*0.7, cy+hs*0.6), color, 2)

	case IconShuffle:
		// Crossed arrows
		canvas.DrawLine(geometry.Pt(cx-hs*0.7, cy-hs*0.4), geometry.Pt(cx+hs*0.5, cy+hs*0.4), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx-hs*0.7, cy+hs*0.4), geometry.Pt(cx+hs*0.5, cy-hs*0.4), color, 1.8)
		// Arrow heads
		canvas.DrawLine(geometry.Pt(cx+hs*0.3, cy-hs*0.55), geometry.Pt(cx+hs*0.55, cy-hs*0.4), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx+hs*0.3, cy+hs*0.55), geometry.Pt(cx+hs*0.55, cy+hs*0.4), color, 1.8)

	case IconRepeat:
		// Loop rectangle
		rRect := geometry.Rect{
			Min: geometry.Pt(cx-hs*0.7, cy-hs*0.45),
			Max: geometry.Pt(cx+hs*0.7, cy+hs*0.45),
		}
		canvas.StrokeRoundRect(rRect, color, 3, 1.8)
		// Arrow head
		canvas.DrawLine(geometry.Pt(cx+hs*0.4, cy-hs*0.65), geometry.Pt(cx+hs*0.7, cy-hs*0.45), color, 1.8)

	case IconVolume:
		// Speaker box & cone
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(cx-hs*0.7, cy-hs*0.3),
			Max: geometry.Pt(cx-hs*0.2, cy+hs*0.3),
		}, color, 1)
		// Cone slant
		canvas.DrawLine(geometry.Pt(cx-hs*0.2, cy-hs*0.3), geometry.Pt(cx+hs*0.1, cy-hs*0.6), color, 2)
		canvas.DrawLine(geometry.Pt(cx-hs*0.2, cy+hs*0.3), geometry.Pt(cx+hs*0.1, cy+hs*0.6), color, 2)
		canvas.DrawLine(geometry.Pt(cx+hs*0.1, cy-hs*0.6), geometry.Pt(cx+hs*0.1, cy+hs*0.6), color, 2)
		// Sound wave arc
		canvas.DrawLine(geometry.Pt(cx+hs*0.4, cy-hs*0.35), geometry.Pt(cx+hs*0.55, cy), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx+hs*0.55, cy), geometry.Pt(cx+hs*0.4, cy+hs*0.35), color, 1.8)

	case IconQueue:
		// 3 horizontal lines
		y0 := cy - hs*0.5
		y1 := cy
		y2 := cy + hs*0.5
		canvas.DrawLine(geometry.Pt(cx-hs*0.7, y0), geometry.Pt(cx+hs*0.7, y0), color, 2)
		canvas.DrawLine(geometry.Pt(cx-hs*0.7, y1), geometry.Pt(cx+hs*0.7, y1), color, 2)
		canvas.DrawLine(geometry.Pt(cx-hs*0.7, y2), geometry.Pt(cx+hs*0.7, y2), color, 2)

	case IconHeart:
		r := hs * 0.4
		// Left lobe
		canvas.StrokeCircle(geometry.Pt(cx-r*0.7, cy-r*0.4), r, color, 1.8)
		// Right lobe
		canvas.StrokeCircle(geometry.Pt(cx+r*0.7, cy-r*0.4), r, color, 1.8)
		// Bottom V
		canvas.DrawLine(geometry.Pt(cx-r*1.4, cy-r*0.1), geometry.Pt(cx, cy+hs*0.8), color, 2)
		canvas.DrawLine(geometry.Pt(cx+r*1.4, cy-r*0.1), geometry.Pt(cx, cy+hs*0.8), color, 2)

	case IconMore:
		// 3 horizontal or vertical dots
		r := float32(2)
		canvas.DrawCircle(geometry.Pt(cx-hs*0.6, cy), r, color)
		canvas.DrawCircle(geometry.Pt(cx, cy), r, color)
		canvas.DrawCircle(geometry.Pt(cx+hs*0.6, cy), r, color)

	case IconLyrics: // Microphone
		w := hs * 0.42
		canvas.StrokeRoundRect(geometry.Rect{
			Min: geometry.Pt(cx-w, cy-hs*0.9),
			Max: geometry.Pt(cx+w, cy+hs*0.05),
		}, color, w, 1.8)
		// Cradle
		canvas.DrawLine(geometry.Pt(cx-hs*0.6, cy-hs*0.1), geometry.Pt(cx-hs*0.6, cy+hs*0.2), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx-hs*0.6, cy+hs*0.2), geometry.Pt(cx+hs*0.6, cy+hs*0.2), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx+hs*0.6, cy-hs*0.1), geometry.Pt(cx+hs*0.6, cy+hs*0.2), color, 1.8)
		// Stand
		canvas.DrawLine(geometry.Pt(cx, cy+hs*0.2), geometry.Pt(cx, cy+hs*0.7), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx-hs*0.35, cy+hs*0.72), geometry.Pt(cx+hs*0.35, cy+hs*0.72), color, 1.8)

	case IconInfo:
		r := hs * 0.8
		canvas.StrokeCircle(geometry.Pt(cx, cy), r, color, 1.8)
		canvas.DrawCircle(geometry.Pt(cx, cy-hs*0.35), 1.6, color)
		canvas.DrawLine(geometry.Pt(cx, cy-hs*0.05), geometry.Pt(cx, cy+hs*0.5), color, 2)

	case IconCheck:
		canvas.DrawLine(geometry.Pt(cx-hs*0.75, cy), geometry.Pt(cx-hs*0.15, cy+hs*0.6), color, 2.2)
		canvas.DrawLine(geometry.Pt(cx-hs*0.15, cy+hs*0.6), geometry.Pt(cx+hs*0.8, cy-hs*0.65), color, 2.2)

	case IconAlert:
		canvas.DrawLine(geometry.Pt(cx, cy-hs*0.85), geometry.Pt(cx-hs*0.85, cy+hs*0.65), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx-hs*0.85, cy+hs*0.65), geometry.Pt(cx+hs*0.85, cy+hs*0.65), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx+hs*0.85, cy+hs*0.65), geometry.Pt(cx, cy-hs*0.85), color, 1.8)
		canvas.DrawLine(geometry.Pt(cx, cy-hs*0.2), geometry.Pt(cx, cy+hs*0.2), color, 2)
		canvas.DrawCircle(geometry.Pt(cx, cy+hs*0.45), 1.5, color)

	case IconWaveform:
		// 4 vertical audio waveform bars
		w := float32(3.5)
		gap := float32(3)
		startX := cx - (4*w+3*gap)/2

		// Bar 1 (medium)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(startX, cy-hs*0.4),
			Max: geometry.Pt(startX+w, cy+hs*0.4),
		}, color, 2)
		// Bar 2 (tallest)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(startX+w+gap, cy-hs*0.9),
			Max: geometry.Pt(startX+w+gap+w, cy+hs*0.9),
		}, color, 2)
		// Bar 3 (medium-high)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(startX+2*(w+gap), cy-hs*0.6),
			Max: geometry.Pt(startX+2*(w+gap)+w, cy+hs*0.6),
		}, color, 2)
		// Bar 4 (medium-low)
		canvas.DrawRoundRect(geometry.Rect{
			Min: geometry.Pt(startX+3*(w+gap), cy-hs*0.35),
			Max: geometry.Pt(startX+3*(w+gap)+w, cy+hs*0.35),
		}, color, 2)
	}
}
