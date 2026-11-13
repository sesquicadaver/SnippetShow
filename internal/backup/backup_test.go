package backup

import "testing"

func TestExportStub(t *testing.T) {
	if err := Export("/tmp/snippetshow-backup.json"); err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
