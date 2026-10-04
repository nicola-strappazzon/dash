package indexes

import (
	"testing"

	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
)

func TestIndexSize(t *testing.T) {
	if got, want := indexSize(mysql.Index{}), "—"; got != want {
		t.Errorf("indexSize() = %q, want %q", got, want)
	}
	if got, want := indexSize(mysql.Index{SizeBytes: 1024, SizeKnown: true}), "1 KiB"; got != want {
		t.Errorf("indexSize() = %q, want %q", got, want)
	}
}
