package app

import (
	"fmt"
	"strings"
)

const lineNumWidthMin = 3

// renderPreview adds line numbers to every line and renders fenced code
// blocks as plain text. No Chroma ANSI theming — terminal-agnostic readability.
//
// ponytail: no full markdown renderer — line scanner with fenced-block
// awareness is the smallest thing that works. Add Chroma/highlighting back
// only if contrast is verified against the specific terminal theme.
func renderPreview(raw string, _width int, style styles, searchTerm string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	lnw := lineWidth(len(lines))
	if lnw < lineNumWidthMin {
		lnw = lineNumWidthMin
	}

	var b strings.Builder
	inCode := false
	var codeLines []string
	codeStart := 0

	flushCode := func() {
		if len(codeLines) > 0 {
			b.WriteString(renderCodeBlock(codeLines, lnw, codeStart, style, searchTerm))
			codeLines = nil
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(line, "```") {
			flushCode()
			inCode = !inCode
			codeStart = i + 2
			ln := fmt.Sprintf("%*d │ ", lnw, i+1)
			b.WriteString(style.dim.Render(ln) + style.dim.Render(line) + "\n")
			continue
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}
		ln := fmt.Sprintf("%*d │ ", lnw, i+1)
		b.WriteString(style.dim.Render(ln) + highlightText(line, searchTerm, style) + "\n")
	}
	flushCode()
	return b.String()
}

func renderCodeBlock(lines []string, lnw int, startLine int, style styles, searchTerm string) string {
	var b strings.Builder
	for i, line := range lines {
		ln := fmt.Sprintf("%*d │ ", lnw, startLine+i)
		b.WriteString(style.dim.Render(ln) + highlightText(line, searchTerm, style) + "\n")
	}
	return b.String()
}

// highlightText wraps occurrences of term in text with style.selected.
// This is contrast-safe because it uses the app's defined selection style
// (inverted/reverse video), not hardcoded ANSI colors.
//
// ponytail: simple case-insensitive scan, no word-boundary logic. Add
// proper token-aware highlighting only if plain-text matching causes false
// positives inside code tokens.
func highlightText(text, term string, style styles) string {
	if term == "" {
		return text
	}
	lower := strings.ToLower(text)
	termLower := strings.ToLower(term)
	matchStyle := style.searchHighlight
	var b strings.Builder
	last := 0
	for last < len(text) {
		idx := strings.Index(lower[last:], termLower)
		if idx < 0 {
			b.WriteString(text[last:])
			break
		}
		pos := last + idx
		b.WriteString(text[last:pos])
		b.WriteString(matchStyle.Render(text[pos : pos+len(term)]))
		last = pos + len(term)
	}
	return b.String()
}

func lineWidth(n int) int {
	if n < 10 {
		return 2
	}
	if n < 100 {
		return 3
	}
	if n < 1000 {
		return 4
	}
	if n < 10000 {
		return 5
	}
	return 6
}
