package agent

import "testing"

func TestNamesWorkspaceFile(t *testing.T) {
	exists := func(path string) bool {
		return path == "calc.go" || path == "docs/guide.md" || path == "ไฟล์.txt"
	}
	cases := map[string]bool{
		"Find and fix the bug in calc.go, then run the tests": true,
		"แก้บั๊กในไฟล์ calc.go ให้หน่อย":                      true,
		"I updated `docs/guide.md`.":                          true,
		"แก้ไข ไฟล์.txt ให้หน่อย":                             true,
		"what is your name?": false,
		"สวัสดี ช่วยอธิบายหน่อย":      false,
		"see missing.go and other.md": false,
		"the version is 1.2.3, done.": false,
		"":                            false,
	}
	for text, want := range cases {
		if got := namesWorkspaceFile(text, exists); got != want {
			t.Errorf("%q = %v, want %v", text, got, want)
		}
	}
	if namesWorkspaceFile("calc.go", nil) {
		t.Error("without a probe nothing is a file")
	}
}
