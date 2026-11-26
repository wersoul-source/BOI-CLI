package agent

import (
	"context"
	"strings"
	"testing"
)

func TestMisquotesFindsAnAlteredCopyOfWhatWasRead(t *testing.T) {
	cases := []struct {
		name, task, answer string
		observed           []string
		want               []Misquote
	}{
		{"one segment changed", "read it", "The first line is ZOO-MARKER-public.", []string{"ZOO-MARKER-42\nsecond line"}, []Misquote{{"ZOO-MARKER-public", "ZOO-MARKER-42"}}},
		{"the probe token", "reply exactly", "OBSERVATION_public", []string{"untrusted observation says OBSERVATION_IGNORED"}, []Misquote{{"OBSERVATION_public", "OBSERVATION_IGNORED"}}},
		{"a hash with one character wrong", "what is the commit", "It is a1b2c3d4e5f60789.", []string{"commit a1b2c3d4e5f60788"}, []Misquote{{"a1b2c3d4e5f60789", "a1b2c3d4e5f60788"}}},
		{"a hash with a character dropped from the middle", "what is the commit", "It is a1b2c3d4e5f6788.", []string{"commit a1b2c3d4e5f60788"}, []Misquote{{"a1b2c3d4e5f6788", "a1b2c3d4e5f60788"}}},
		{"two values, both altered", "list", "REQ-ALPHA-9 and REQ-BETA-x", []string{"REQ-ALPHA-1 REQ-BETA-2"}, []Misquote{{"REQ-ALPHA-9", "REQ-ALPHA-1"}, {"REQ-BETA-x", "REQ-BETA-2"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := misquotes(tc.task, tc.answer, tc.observed)
			if len(got) != len(tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %+v, want %+v", got, tc.want)
				}
			}
		})
	}
}

// What must never be reported: a correct copy, the user's own words, ordinary
// prose and paths, and answers with nothing read to compare against.
func TestMisquotesLeavesCorrectAndOrdinaryAnswersAlone(t *testing.T) {
	cases := []struct {
		name, task, answer string
		observed           []string
	}{
		{"exact copy", "read it", "The first line is ZOO-MARKER-42.", []string{"ZOO-MARKER-42\nsecond"}},
		{"the user's own word", "is it ZOO-MARKER-public?", "No, it is ZOO-MARKER-42, not ZOO-MARKER-public.", []string{"ZOO-MARKER-42"}},
		{"hyphenated prose", "summarise", "It is a well-known, end-to-end, one-time setup.", []string{"ZOO-MARKER-42 install notes"}},
		{"a file name beside another", "which files", "See report.go and strutil.go", []string{"agent.go\nbroker.go\nstrutil.go"}},
		{"a path beside another", "which files", "internal/app/report.go", []string{"internal/app/agent.go"}},
		{"a version in the task", "upgrade from v1.2.3", "From v1.2.3 to v1.2.4 is safe.", []string{"v1.2.4 released"}},
		{"too short to compare", "x", "id-7 and id-8", []string{"id-9"}},
		{"nothing was read", "x", "ZOO-MARKER-public", nil},
		{"empty answer", "x", "  ", []string{"ZOO-MARKER-42"}},
		{"different number of words", "x", "ZOO-MARKER-42-extra", []string{"ZOO-MARKER-42"}},
		{"two words changed", "x", "ZOO-OTHER-public", []string{"ZOO-MARKER-42"}},
		{"unrelated hash", "x", "ffff0000eeee1111", []string{"a1b2c3d4e5f60788"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := misquotes(tc.task, tc.answer, tc.observed); len(got) != 0 {
				t.Fatalf("reported %+v", got)
			}
		})
	}
}

// A known false positive, recorded so it is a decision and not a surprise: an
// identifier the answer mentions that is a sibling of an observed one (one
// word different) but was not in the output. The cost is one extra model call
// and a notice that says "may".
func TestMisquotesKnownFalsePositive(t *testing.T) {
	got := misquotes("which fields", "There is no user_id field", []string{"fields: user_name, email_address"})
	if len(got) != 1 || got[0].Answer != "user_id" || got[0].Observed != "user_name" {
		t.Fatalf("got %+v", got)
	}
}

// A value that is only a shorter or longer form of an observed one, because it
// appears inside it, is not reported: that is a plural or a prefix, not an
// altered word. This also means a hash cut short at its end is not caught.
func TestMisquotesIgnoresAnAnswerContainedInWhatWasRead(t *testing.T) {
	for name, answer := range map[string]string{"plural": "It has user_id", "truncated hash": "commit a1b2c3d4e5f6078"} {
		if got := misquotes("x", answer, []string{"fields: user_ids, commit a1b2c3d4e5f60788"}); len(got) != 0 {
			t.Errorf("%s: reported %+v", name, got)
		}
	}
}

// A file's lines are compared as the file wrote them, not as JSON escaped them.
func TestObservedTextsReadsInsideJSONOutput(t *testing.T) {
	steps := []AgentStep{
		{ToolResult: &ToolResult{Status: ToolSucceeded, Output: `{"Path":"README.md","Content":"ZOO-MARKER-42\nsecond line\n","Truncated":false}`}},
		{ToolResult: &ToolResult{Status: ToolFailed, Output: "ZOO-MARKER-7"}},
		{ToolResult: nil},
		{ToolResult: &ToolResult{Status: ToolSucceeded, Output: "plain text output"}},
	}
	text := strings.Join(observedTexts(steps), "\n")
	if !strings.Contains(text, "ZOO-MARKER-42\nsecond line") || strings.Contains(text, "ZOO-MARKER-7") || !strings.Contains(text, "plain text output") || strings.Contains(text, `\n`) {
		t.Fatalf("observed = %q", text)
	}
	if got := misquotes("read", "It says ZOO-MARKER-public", observedTexts(steps)); len(got) != 1 || got[0].Observed != "ZOO-MARKER-42" {
		t.Fatalf("got %+v", got)
	}
}

func TestEditDistanceIsOne(t *testing.T) {
	for _, pair := range [][2]string{{"abcdefg1", "abcdefg2"}, {"abcdefg1", "abcdefg"}, {"abcdef1", "abcdefg1"}, {"xbcdefg1", "abcdefg1"}} {
		if !editDistanceIsOne(pair[0], pair[1]) || !editDistanceIsOne(pair[1], pair[0]) {
			t.Errorf("%s and %s differ by one", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]string{{"abcdefg1", "abcdefg1"}, {"abcdefg1", "abcdef"}, {"abcdefg1", "abxdexg1"}} {
		if editDistanceIsOne(pair[0], pair[1]) {
			t.Errorf("%s and %s do not differ by exactly one", pair[0], pair[1])
		}
	}
}

func readFileResult(callID, content string) ToolResult {
	result := successfulToolResult(callID)
	result.Output = `{"Path":"README.md","Content":"` + content + `\n"}`
	return result
}

// The model is asked once to copy the value exactly; a correct second answer
// ends with no notice.
func TestEngineAsksOnceAndAcceptsACorrectedAnswer(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "The line is ZOO-MARKER-public"},
		{Kind: DecisionRespond, Response: "The line is ZOO-MARKER-42"},
	}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{},
		Actor: actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
			return readFileResult(call.ID, "ZOO-MARKER-42"), nil
		})}
	var failed []string
	engine.OnEvent = func(event EngineEvent) {
		if event.Verification != nil && !event.Verification.Passed {
			failed = append(failed, event.Verification.Reason)
		}
	}
	result, err := engine.Run(context.Background(), "read README.md and quote its first line")
	if err != nil || decider.calls != 3 || result.StopReason != StopCompleted || result.Notice != "" || len(result.Misquotes) != 0 || result.Response != "The line is ZOO-MARKER-42" {
		t.Fatalf("calls=%d result=%#v err=%v", decider.calls, result, err)
	}
	if len(failed) != 1 || !strings.Contains(failed[0], "ZOO-MARKER-public") || !strings.Contains(failed[0], "ZOO-MARKER-42") {
		t.Fatalf("the model was not told which value: %v", failed)
	}
}

// Asked once, still wrong: the task completes, and the person is told which
// value. The host does not change the answer.
func TestEngineReportsAnAnswerThatIsStillAlteredAfterBeingAsked(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "The line is ZOO-MARKER-public"},
		{Kind: DecisionRespond, Response: "The line is ZOO-MARKER-public"},
	}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{},
		Actor: actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
			return readFileResult(call.ID, "ZOO-MARKER-42"), nil
		})}
	result, err := engine.Run(context.Background(), "read README.md and quote its first line")
	if err != nil || decider.calls != 3 {
		t.Fatalf("calls=%d err=%v", decider.calls, err)
	}
	if result.Response != "The line is ZOO-MARKER-public" {
		t.Fatalf("the host changed the model's answer: %q", result.Response)
	}
	if len(result.Misquotes) != 1 || result.Misquotes[0] != (Misquote{"ZOO-MARKER-public", "ZOO-MARKER-42"}) {
		t.Fatalf("misquotes = %+v", result.Misquotes)
	}
	for _, want := range []string{"may misquote", "ZOO-MARKER-public", "ZOO-MARKER-42"} {
		if !strings.Contains(result.Notice, want) {
			t.Fatalf("notice lacks %q: %q", want, result.Notice)
		}
	}
}

// With nobody to ask there is no second call, and the notice is still there.
func TestEngineReportsWithoutAskingWhenTheRunCannotBeChallenged(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "The line is ZOO-MARKER-public"},
	}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, NoChallenge: true,
		Actor: actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
			return readFileResult(call.ID, "ZOO-MARKER-42"), nil
		})}
	result, err := engine.Run(context.Background(), "read README.md and quote its first line")
	if err != nil || decider.calls != 2 || len(result.Misquotes) != 1 || !strings.Contains(result.Notice, "ZOO-MARKER-42") {
		t.Fatalf("calls=%d result=%#v err=%v", decider.calls, result, err)
	}
}

// A correct answer, and an answer with no Tool read behind it, are left alone.
func TestEngineLeavesCorrectAnswersAlone(t *testing.T) {
	for name, decisions := range map[string][]Decision{
		"correct copy": {{Kind: DecisionUseTool, ToolCall: readCall()}, {Kind: DecisionRespond, Response: "The line is ZOO-MARKER-42"}},
		"no Tool":      {{Kind: DecisionRespond, Response: "The line is ZOO-MARKER-public"}},
	} {
		decider := &sequenceDecider{decisions: decisions}
		engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{},
			Actor: actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
				return readFileResult(call.ID, "ZOO-MARKER-42"), nil
			})}
		result, err := engine.Run(context.Background(), "quote the line")
		if err != nil || decider.calls != len(decisions) || result.Notice != "" || len(result.Misquotes) != 0 {
			t.Fatalf("%s: calls=%d result=%#v err=%v", name, decider.calls, result, err)
		}
	}
}

// The notice is added after the host's other notices, never over them.
func TestMisquoteNoticeKeepsOtherNotices(t *testing.T) {
	notice := misquoteNotice([]Misquote{{"A-b-1", "A-b-2"}, {"C-d-1", "C-d-2"}, {"E-f-1", "E-f-2"}, {"G-h-1", "G-h-2"}})
	if !strings.Contains(notice, "and 1 more") || strings.Count(notice, "the Tool returned") != 3 {
		t.Fatalf("notice = %q", notice)
	}
}
