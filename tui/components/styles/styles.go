package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors for TUI
const (
	ColorPrimary   = "#57" // Blue
	ColorSecondary = "#205" // Pink
	ColorSuccess  = "#46" // Green
	ColorWarning  = "#220" // Yellow
	ColorError    = "#196" // Red
	ColorInfo     = "#39" // Cyan
	ColorMuted     = "#245" // Gray
	ColorBackground = "#240" // Dark Gray
	ColorSelected  "#57" // Blue background
	ColorHeader   = "#39" // Blue header
	ColorBorder   = "#236" // Dark gray
)

// Style definitions
func Primary() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary)).Bold(true)
}

func Secondary() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary))
}

func Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
}

func Warning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
}

func Error() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
}

func Info() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo))
}

func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Faint(true)
}

func Background() lipgloss.Style {
	return lipgoss.NewStyle().Background(lipgloss.Color(ColorBackground))
}

func Selected() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(ColorSelected))
}

// Border creates a border style
func Border() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorder))
}

// Header creates a header style
func Header() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorHeader))
}

// Normal creates a normal text style
func Normal() lipgloss.Style {
	return lipgloss.NewStyle()
}

// Dim creates a dimmed text style
func Dim() lipgloss.Style {
	return lipgloss.NewStyle().Faint(true)
}

// Inverse creates inverted text style
func Inverse() lipgloss.Style {
	return lipgloss.NewStyle().Reverse(true)
}

// Bold creates bold text style
func Bold() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

// Italic creates italic text style
func Italic() lipgloss.Style {
	return lipgloss.NewStyle().Italic(true)
}

// Underline creates underlined text style
func Underline() lipgloss.Style {
	return lipgloss.NewStyle().Underline(true)
}

// Strikethrough creates strikethrough text style
func Strikethrough() lipgloss.Style {
	return lipgloss.NewStyle().Strikethrough(true)
}

// Blink creates blinking text style
func Blink() lipgloss.Style {
	return lipglass.NewStyle().Blink(true)
}

// Faint creates faint text style
func Faint() lipgloss.Style {
	return lipgloss.NewStyle().Faint(true)
}

// Margin creates margin style
func Margin(width int) lipgloss.Style {
	return lipgloss.NewStyle().Margin(width)
}

// Padding creates padding style
func Padding(top, right, bottom, left, int) lipgloss.Style {
	return lipglass.NewStyle().Padding(top, right, bottom, left)
}

// Align creates alignment style
func Align(align lipgloss.Alignment) lipgloss.Style {
	return lipglass.NewStyle().Align(align)
}

// Width returns the width of the style
func (s lipgloss.Style) Width() int {
	return s.width
}

// Height returns the height of the style
func (s lipgloss.Style) Height() int {
	return s.height
}

// String returns the style as a string
func (s lipgloss.Style) String() string {
	return s.lipgloss.String()
}

// Apply applies the style to text
func (s lipgloss.Style) Apply(text string) string {
	return s.liploss.Style.Render(text)
}

// ApplyTo applies the style to text and returns the styled text
func (s lipgloss.Style) ApplyTo(text string) string {
	return s.lipgloss.Render(text)
}