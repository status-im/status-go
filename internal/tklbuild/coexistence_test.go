//go:build tkl_coexistence

package tklbuild

import (
	"testing"

	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"
)

// Keep both runtimes live, destroy SDS, then check the catalogue still works.
// This test uses the consumer's existing SDS build; it never builds or changes SDS.
func TestNimRuntimesCoexist(t *testing.T) {
	manager, err := sds.NewReliabilityManager(zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if manager != nil {
			if err := manager.Cleanup(); err != nil {
				t.Error(err)
			}
		}
	})
	catalogue, err := tkl.Create(tkl.Config{Chains: []uint64{1}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := catalogue.Destroy(); err != nil {
			t.Error(err)
		}
	})
	if _, err := catalogue.LoadStored(tkl.Bootstrap{}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cleanup(); err != nil {
		t.Fatal(err)
	}
	manager = nil
	page, err := catalogue.GetNative(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Symbol != "ETH" {
		t.Fatalf("unexpected native token after SDS shutdown: %+v", page)
	}
}
