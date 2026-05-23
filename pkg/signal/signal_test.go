package signal_test

import (
	"testing"

	"github.com/vugra/vugra/pkg/signal"
)

func TestIntZeroValueSignal(t *testing.T) {
	var count signal.Int
	if count.Get() != 0 {
		t.Fatalf("initial count = %d", count.Get())
	}
	count.Set(2)
	count.Update(func(value int) int { return value + 3 })
	if count.Get() != 5 {
		t.Fatalf("updated count = %d", count.Get())
	}
}

func TestBoolAndStringSignals(t *testing.T) {
	enabled := signal.NewBool(true)
	if !enabled.Get() {
		t.Fatal("enabled should be true")
	}
	name := signal.NewString("vugra")
	if name.Get() != "vugra" {
		t.Fatalf("name = %q", name.Get())
	}
}

func TestGenericSignal(t *testing.T) {
	items := signal.New([]string{"alpha"})
	items.Update(func(values []string) []string {
		return append(values, "beta")
	})
	if got := items.Get(); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("items = %#v", got)
	}
	items.SetAny([]string{"gamma"})
	if got := items.Get(); len(got) != 1 || got[0] != "gamma" {
		t.Fatalf("items after SetAny = %#v", got)
	}
	items.SetAny("ignored")
	if got := items.Get(); len(got) != 1 || got[0] != "gamma" {
		t.Fatalf("items after mismatched SetAny = %#v", got)
	}
}
