package manparser

import "testing"

func TestSummaryStub(t *testing.T) {
	_, err := Summary("ls")
	if err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
