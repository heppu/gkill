package killer

import (
	"testing"
)

func TestNewKiller(t *testing.T) {
	if _, err := NewKiller(""); err != nil {
		t.Fatal(err)
	}
}
