package terminal

import (
	"strings"

	"github.com/rivo/uniseg"
)

// StringWidth returns how many terminal columns s occupies, ignoring any ANSI
// escape sequences in it.
//
// It measures grapheme clusters, not runes, which is the only unit a terminal
// actually draws: a Thai syllable and its vowel and tone marks are one cell,
// an emoji is two, and a ZWJ sequence such as a family emoji is still two.
// Counting runes instead broke the box drawn around every message by one
// column per emoji.
func StringWidth(s string) int {
	return uniseg.StringWidth(StripANSI(s))
}

// Wrap breaks s into lines of at most width columns. It cuts between grapheme
// clusters, never inside one, so a Thai syllable or an emoji sequence stays
// whole. A line that holds a space breaks at the last one before the limit, so
// Latin words stay whole; Thai writes without word spaces and wraps on width
// alone, which is what a Thai reader expects.
//
// A single cluster wider than width is emitted on its own line rather than
// split, because splitting it would corrupt the character.
func Wrap(s string, width int) []string {
	if width <= 0 {
		return strings.Split(s, "\n")
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		lines = append(lines, wrapLine(paragraph, width)...)
	}
	return lines
}

type cluster struct {
	text  string
	width int
}

func clusters(s string) []cluster {
	out := make([]cluster, 0, len(s))
	graphemes := uniseg.NewGraphemes(s)
	for graphemes.Next() {
		out = append(out, cluster{text: graphemes.Str(), width: graphemes.Width()})
	}
	return out
}

func wrapLine(line string, width int) []string {
	if StringWidth(line) <= width {
		return []string{line}
	}
	items := clusters(StripANSI(line))
	var lines []string
	start, lineWidth, lastSpace := 0, 0, -1
	for index, item := range items {
		if item.text == " " {
			lastSpace = index
		}
		if lineWidth+item.width > width && index > start {
			cut := index
			if lastSpace > start {
				cut = lastSpace
			}
			lines = append(lines, join(items[start:cut]))
			start = cut
			for start < len(items) && items[start].text == " " {
				start++
			}
			lastSpace, lineWidth = -1, 0
			for back := start; back <= index && back < len(items); back++ {
				lineWidth += items[back].width
			}
			continue
		}
		lineWidth += item.width
	}
	if start < len(items) {
		lines = append(lines, join(items[start:]))
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func join(items []cluster) string {
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString(item.text)
	}
	return builder.String()
}
