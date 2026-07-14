package orm

import (
	"context"
	"testing"
)

func TestWithFullScanIsExplicit(t *testing.T) {
	if FullScanAllowed(context.Background()) {
		t.Fatal("background allows full scan")
	}
	if !FullScanAllowed(WithFullScan(context.Background())) {
		t.Fatal("marker not preserved")
	}
	if !FullScanAllowed(WithFullScan(nil)) {
		t.Fatal("nil context marker not preserved")
	}
}
