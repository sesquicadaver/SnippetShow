package clipboard

import "testing"

func TestCopyStub(t *testing.T) {
	if err := Copy("echo ok"); err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
