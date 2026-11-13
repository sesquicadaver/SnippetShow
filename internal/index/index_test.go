package index

import "testing"

func TestUpdateIndexStub(t *testing.T) {
	if err := (&Updater{}).UpdateIndex(); err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
