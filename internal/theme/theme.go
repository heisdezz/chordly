package theme

import (
	"github.com/gogpu/ui/widget"
)

// Palette contains the color tokens derived from the Metrolist reference design.
var (
	// Backgrounds
	ColorCanvas        = widget.Hex(0x0C1216) // Deep dark teal/slate window background
	ColorSidebar       = widget.Hex(0x0E1419) // Left rail background
	ColorSurface       = widget.Hex(0x131B21) // Card and container surface
	ColorSurfaceLight  = widget.Hex(0x1A252D) // Elevated cards / hover state
	ColorSurfaceSubtle = widget.Hex(0x10171D) // Topbar search input background
	ColorPlayerBar     = widget.Hex(0x0F161C) // Bottom floating player dock

	// Accent / Mint
	ColorAccent      = widget.Hex(0x5CE1B6) // Iconic mint green from reference image
	ColorAccentHover = widget.Hex(0x4FD4A9) // Slightly deeper mint on hover
	ColorAccentSoft  = widget.Hex(0x18372D) // Soft translucent mint for active item background
	ColorOnAccent    = widget.Hex(0x091D16) // Dark readable text on mint pill buttons

	// Borders & Dividers
	ColorBorder      = widget.Hex(0x1D2B35) // Subtle sleek borders
	ColorBorderMuted = widget.Hex(0x16222B)

	// Typography
	ColorTextPrimary   = widget.Hex(0xF3F6F8) // Pure crisp white/slate
	ColorTextSecondary = widget.Hex(0x9EB0BD) // Secondary metadata text
	ColorTextMuted     = widget.Hex(0x6E808D) // Dimmed icons and timestamps

	// UI Action States
	ColorHeartActive   = widget.Hex(0xEF4444) // Liked track red/coral
	ColorScrubberTrack = widget.Hex(0x22323D) // Unfilled audio progress bar
	ColorCardOverlay   = widget.Hex(0x18242C) // Semi-transparent card sheen
)
