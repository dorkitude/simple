package tui

import (
	"fmt"
	"strings"
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func windowRange(total, selected, capacity int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if capacity <= 0 || total <= capacity {
		return 0, total
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}

	start := selected - capacity/2
	if start < 0 {
		start = 0
	}
	end := start + capacity
	if end > total {
		end = total
		start = end - capacity
	}
	return start, end
}

func clipLines(lines []string, maxLines int) []string {
	if maxLines <= 0 {
		return []string{}
	}
	if len(lines) <= maxLines {
		return lines
	}
	hidden := len(lines) - maxLines + 1
	out := append([]string{}, lines[:maxLines-1]...)
	out = append(out, subtitleStyle.Render(fmt.Sprintf("... (%d more lines)", hidden)))
	return out
}

func clipMultilineText(s string, maxLines int) string {
	if s == "" {
		return ""
	}
	return strings.Join(clipLines(strings.Split(s, "\n"), maxLines), "\n")
}

func wrapLabelValue(label, value string, width int) []string {
	if width <= 0 {
		width = 80
	}
	chunks := hardWrap(value, width)
	if len(chunks) == 0 {
		return []string{label}
	}
	lines := []string{label + chunks[0]}
	indent := strings.Repeat(" ", len(label))
	for _, c := range chunks[1:] {
		lines = append(lines, indent+c)
	}
	return lines
}

func hardWrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	if s == "" {
		return []string{""}
	}
	r := []rune(s)
	out := make([]string, 0, (len(r)/width)+1)
	for len(r) > 0 {
		n := minInt(width, len(r))
		out = append(out, string(r[:n]))
		r = r[n:]
	}
	return out
}
