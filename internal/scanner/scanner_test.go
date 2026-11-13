package scanner

import "testing"

func TestDetectPMStub(t *testing.T) {
	_, err := DetectPM()
	if err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
