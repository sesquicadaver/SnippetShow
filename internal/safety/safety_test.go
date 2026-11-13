package safety

import "testing"

func TestAnalyzeStub(t *testing.T) {
	_, err := Analyze("rm -rf /")
	if err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
