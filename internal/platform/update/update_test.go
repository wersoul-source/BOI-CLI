package update

import "testing"

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal", a: "0.3.0", b: "0.3.0", want: 0},
		{name: "newer patch", a: "0.3.1", b: "0.3.0", want: 1},
		{name: "older minor", a: "0.2.9", b: "0.3.0", want: -1},
		{name: "missing patch", a: "1.0", b: "1.0.0", want: 0},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := CompareVersions(testCase.a, testCase.b); got != testCase.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", testCase.a, testCase.b, got, testCase.want)
			}
		})
	}
}
