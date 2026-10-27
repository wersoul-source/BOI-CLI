package tui

import (
	"strings"
	"testing"

	term "github.com/boi-family/boi-cli/internal/platform/terminal"
)

func widestLine(s string) (int, string) {
	widest, found := 0, ""
	for _, line := range strings.Split(s, "\n") {
		if w := term.StringWidth(line); w > widest {
			widest, found = w, line
		}
	}
	return widest, found
}

// A message can arrive before the first WindowSizeMsg, when the width is still
// zero. The box width then went negative and strings.Repeat panicked.
func TestChatRendersBeforeTheFirstResize(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("rendering at width 0 panicked: %v", r)
		}
	}()
	chat := NewChat()
	chat.AddMessage("user", "สวัสดีครับ 🙏")
	chat.AddAgentMessage("ยินดีครับ", "maxplus", "deepseek-v4.1-flash", 42)
	if chat.View() == "" {
		t.Fatal("nothing rendered")
	}
}

// Nothing may be wider than the terminal. A long line used to be written whole,
// so the terminal wrapped it and the right border landed in the middle of the
// next row.
func TestChatNeverDrawsWiderThanTheTerminal(t *testing.T) {
	for _, width := range []int{143, 80, 51, 30, 20, 10} {
		for name, content := range map[string]string{
			"thai":  strings.Repeat("สวัสดีครับ", 40),
			"latin": strings.Repeat("the quick brown fox ", 20),
			"emoji": strings.Repeat("🙏 ok ", 30),
			"mixed": "รายงาน 📋 report.json เสร็จแล้วครับ " + strings.Repeat("ก", 120),
		} {
			chat := NewChat()
			chat.SetSize(width, 20)
			chat.AddMessage("user", content)
			if got, line := widestLine(chat.viewport.View()); got > width {
				t.Errorf("width %d, %s: line is %d columns: %q", width, name, got, line)
			}
		}
	}
}

// Every row of one box must be the same width, emoji or not, or the border
// zig-zags down the screen.
func TestChatBoxRowsAreAllTheSameWidth(t *testing.T) {
	chat := NewChat()
	chat.SetSize(60, 20)
	chat.AddMessage("user", "plain line\nwith emoji 🙏 here\nไทย ก่อน")
	seen := map[int]bool{}
	for _, line := range strings.Split(chat.viewport.View(), "\n") {
		if strings.HasPrefix(line, "│") {
			seen[term.StringWidth(line)] = true
		}
	}
	if len(seen) != 1 {
		t.Fatalf("box rows have differing widths: %v", seen)
	}
}

// Boxes are drawn to a fixed width, so a resize has to redraw them.
func TestChatReflowsOnResize(t *testing.T) {
	chat := NewChat()
	chat.SetSize(120, 20)
	chat.AddMessage("user", strings.Repeat("สวัสดี", 30))
	chat.SetSize(44, 20)
	if got, line := widestLine(chat.viewport.View()); got > 44 {
		t.Fatalf("after resize the box is still %d columns: %q", got, line)
	}
	chat.SetSize(120, 20)
	if got, _ := widestLine(chat.viewport.View()); got > 120 {
		t.Fatalf("after widening the box is %d columns", got)
	}
}

// A reply being written is shown as it arrives and is replaced by the real
// message when the turn ends.
func TestChatShowsAndThenReplacesTheLiveReply(t *testing.T) {
	chat := NewChat()
	chat.SetSize(60, 20)
	chat.AddMessage("user", "สวัสดี")
	chat.SetPartial("กำลัง")
	chat.SetPartial("กำลังคิดอยู่ครับ")
	view := chat.viewport.View()
	if !strings.Contains(view, "กำลังคิดอยู่ครับ") {
		t.Fatal("the live reply is not shown")
	}
	if strings.Contains(view, "กำลัง\n") && strings.Count(view, "writing") != 1 {
		t.Fatal("cumulative updates must replace, not stack")
	}
	chat.AddAgentMessage("เสร็จแล้วครับ", "maxplus", "m", 10)
	view = chat.viewport.View()
	if strings.Contains(view, "writing") {
		t.Fatal("the live bubble outlived the turn")
	}
	if !strings.Contains(view, "เสร็จแล้วครับ") {
		t.Fatal("the final message is missing")
	}
}

// A live reply must obey the same width rules as a finished one.
func TestChatLiveReplyStaysInsideTheTerminal(t *testing.T) {
	chat := NewChat()
	chat.SetSize(51, 20)
	chat.SetPartial(strings.Repeat("สวัสดีครับ 🙏 ", 30))
	if got, line := widestLine(chat.viewport.View()); got > 51 {
		t.Fatalf("live reply is %d columns: %q", got, line)
	}
}
