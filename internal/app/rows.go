package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// rowCell is one column in a table-style row.
type rowCell struct {
	Text  string
	Width int          // 0 = auto
	Style lipgloss.Style
}

// renderCells joins cells into one properly aligned row. Cells beyond
// available width are truncated with "…".
func renderCells(_width int, cells ...rowCell) string {
	if len(cells) == 0 {
		return ""
	}
	// Calculate auto widths from remaining space.
	autoCount := 0
	fixedWidth := 0
	for _, c := range cells {
		if c.Width > 0 {
			fixedWidth += c.Width + 1 // space separator
		} else {
			autoCount++
		}
	}
	autoWidth := 0
	if autoCount > 0 {
		remaining := _width - fixedWidth - 1
		if remaining < 10 {
			remaining = 10
		}
		autoWidth = remaining / autoCount
	}

	var parts []string
	for _, c := range cells {
		w := c.Width
		if w <= 0 {
			w = autoWidth
		}
		t := c.Text
		if visibleWidth(t) > w {
			t = truncateText(t, w)
		}
		rendered := c.Style.Render(t)
		// Pad with spaces to reach requested width.
		padding := w - visibleWidth(t)
		if padding > 0 {
			rendered += strings.Repeat(" ", padding)
		}
		parts = append(parts, rendered)
	}
	return strings.Join(parts, " ")
}

func truncateText(s string, max int) string {
	w := visibleWidth(s)
	if w <= max {
		return s
	}
	runes := []rune(s)
	n := len(runes)
	for n > 0 && visibleWidth(string(runes[:n]))+1 > max {
		n--
	}
	if n < 1 {
		return ""
	}
	return string(runes[:n]) + "…"
}

func visibleWidth(s string) int {
	// lipgloss.Width accounts for ANSI codes.
	return lipgloss.Width(s)
}

// numberCell formats a number with right-alignment padding.
func numberCell(n int, maxDigits int) rowCell {
	return rowCell{
		Text:  fmt.Sprintf("%d", n),
		Width: maxDigits,
	}
}

// renderListLine renders a one-line item with optional cursor prefix.
func renderListLine(width int, selected bool, cells ...rowCell) string {
	prefix := "  "
	if selected {
		prefix = "❯ "
	}
	if len(cells) == 0 {
		return prefix
	}
	cells[0].Text = prefix + cells[0].Text
	return renderCells(width, cells...)
}

// renderBlockRow renders a two-line item: title line with cursor, then meta line.
func renderBlockRow(width int, selected bool, title string, meta []string, desc string, titleStyle, metaStyle lipgloss.Style) string {
	prefix := "  "
	if selected {
		prefix = "❯ "
	}
	var b strings.Builder
	maxTitle := max(10, width-4)
	b.WriteString(titleStyle.Render(prefix + truncateText(title, maxTitle)))
	b.WriteString("\n")
	if len(meta) > 0 {
		b.WriteString(metaStyle.Render("  " + truncateText(strings.Join(meta, " • "), max(10, width-4))))
		b.WriteString("\n")
	}
	if desc != "" {
		b.WriteString(metaStyle.Render("  " + truncateText(desc, max(10, width-4))))
		b.WriteString("\n")
	}
	return b.String()
}

func clampIndex(i, n int) int {
	if n <= 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func clampOffset(sel, offset, total, visible int) int {
	if total <= 0 || visible <= 0 || visible >= total {
		return 0
	}
	if sel < offset {
		return sel
	}
	if sel >= offset+visible {
		return sel - visible + 1
	}
	return offset
}

func rowStyle(style styles, selected bool) lipgloss.Style {
	if selected {
		return style.rowSelected
	}
	return style.rowMuted
}
