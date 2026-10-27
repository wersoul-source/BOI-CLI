package terminal

import (
	"strings"
	"testing"
)

// Thai vowels and tone marks are combining marks the terminal draws on top of
// the consonant before them: they take no column of their own.
func TestStringWidthCountsThaiMarksAsZero(t *testing.T) {
	for _, item := range []struct {
		input string
		want  int
	}{
		{"ก", 1},
		{"ก่อน", 3},
		{"สวัสดี", 4},
		{"สวัสดีครับ", 7},
	} {
		if got := StringWidth(item.input); got != item.want {
			t.Errorf("StringWidth(%q) = %d, want %d", item.input, got, item.want)
		}
	}
}

// An emoji takes two columns. Counting it as one shifted the right border of
// every message box one column left for each emoji in the line.
func TestStringWidthCountsEmojiAsTwo(t *testing.T) {
	for _, input := range []string{"🙏", "📋", "✅", "你", "あ", "、", "👨‍👩‍👧"} {
		if got := StringWidth(input); got != 2 {
			t.Errorf("StringWidth(%q) = %d, want 2", input, got)
		}
	}
}

func TestStringWidthIgnoresANSI(t *testing.T) {
	if got := StringWidth("\x1b[31mred\x1b[0m"); got != 3 {
		t.Errorf("StringWidth with ANSI = %d, want 3", got)
	}
}

// Thai writes without word spaces, so it can only wrap on width. Every line
// must stay inside the limit or the box around it breaks.
func TestWrapKeepsEveryLineInsideTheWidth(t *testing.T) {
	for _, input := range []string{
		strings.Repeat("ก", 300),
		strings.Repeat("สวัสดีครับ", 40),
		strings.Repeat("🙏", 50),
		strings.Repeat("the quick brown fox ", 20),
	} {
		for _, line := range Wrap(input, 24) {
			if got := StringWidth(line); got > 24 {
				t.Fatalf("wrapped line is %d columns wide: %q", got, line)
			}
		}
	}
}

// Wrapping must not lose or reorder anything.
func TestWrapPreservesContent(t *testing.T) {
	input := "สวัสดีครับ ผมชื่อด่าง 🙏 ยินดีที่ได้รู้จัก"
	joined := strings.Join(Wrap(input, 10), "")
	if strings.ReplaceAll(joined, " ", "") != strings.ReplaceAll(input, " ", "") {
		t.Fatalf("content changed:\n got %q\nwant %q", joined, input)
	}
}

// A break falls on a space when there is one, so Latin words stay whole.
func TestWrapBreaksOnSpacesWhenItCan(t *testing.T) {
	for _, line := range Wrap("alpha beta gamma delta epsilon", 12) {
		for _, word := range strings.Fields(line) {
			if !strings.Contains("alpha beta gamma delta epsilon", word) {
				t.Fatalf("a word was split: %q", word)
			}
		}
	}
}

// A cluster wider than the whole width cannot be split without corrupting it.
func TestWrapDoesNotSplitAClusterWiderThanTheLimit(t *testing.T) {
	lines := Wrap("🙏🙏🙏", 1)
	if len(lines) != 3 {
		t.Fatalf("want one emoji per line, got %q", lines)
	}
}

func TestWrapHandlesNonPositiveWidth(t *testing.T) {
	if got := Wrap("a\nb", 0); len(got) != 2 {
		t.Fatalf("want the input lines back, got %q", got)
	}
}
