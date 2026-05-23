package reactivity

import "testing"

func TestSignalGetSetUpdateSubscribe(t *testing.T) {
	scheduler := NewScheduler()
	count := NewWithScheduler(1, scheduler)

	calls := 0
	unsubscribe := count.Subscribe(func() {
		calls++
	})

	if count.Get() != 1 {
		t.Fatalf("initial value = %d", count.Get())
	}
	count.Set(2)
	if calls != 0 {
		t.Fatalf("subscriber ran before flush")
	}
	scheduler.Flush()
	if calls != 1 {
		t.Fatalf("subscriber calls after set = %d", calls)
	}

	count.Update(func(value int) int { return value + 3 })
	scheduler.Flush()
	if count.Get() != 5 {
		t.Fatalf("updated value = %d", count.Get())
	}
	if calls != 2 {
		t.Fatalf("subscriber calls after update = %d", calls)
	}

	unsubscribe()
	count.Set(6)
	scheduler.Flush()
	if calls != 2 {
		t.Fatalf("subscriber ran after unsubscribe: %d", calls)
	}
}

func TestEffectTracksGetDependencies(t *testing.T) {
	scheduler := NewScheduler()
	count := NewWithScheduler(0, scheduler)

	runs := 0
	observed := -1
	NewEffectWithScheduler(func() {
		runs++
		observed = count.Get()
	}, scheduler)

	if runs != 1 || observed != 0 {
		t.Fatalf("initial effect runs=%d observed=%d", runs, observed)
	}
	count.Set(1)
	if runs != 1 {
		t.Fatalf("effect ran before flush")
	}
	scheduler.Flush()
	if runs != 2 || observed != 1 {
		t.Fatalf("after flush runs=%d observed=%d", runs, observed)
	}
}

func TestMultipleSetsBatchIntoOneEffectRun(t *testing.T) {
	scheduler := NewScheduler()
	count := NewWithScheduler(0, scheduler)

	runs := 0
	NewEffectWithScheduler(func() {
		runs++
		_ = count.Get()
	}, scheduler)

	count.Set(1)
	count.Set(2)
	count.Set(3)
	if scheduler.Pending() != 1 {
		t.Fatalf("pending jobs = %d", scheduler.Pending())
	}
	scheduler.Flush()
	if runs != 2 {
		t.Fatalf("effect runs = %d", runs)
	}
}

func TestMultipleEffectsOnSameSignalAllRun(t *testing.T) {
	scheduler := NewScheduler()
	count := NewWithScheduler(0, scheduler)

	firstRuns := 0
	secondRuns := 0
	NewEffectWithScheduler(func() {
		firstRuns++
		_ = count.Get()
	}, scheduler)
	NewEffectWithScheduler(func() {
		secondRuns++
		_ = count.Get()
	}, scheduler)

	count.Set(1)
	scheduler.Flush()
	if firstRuns != 2 || secondRuns != 2 {
		t.Fatalf("runs first=%d second=%d", firstRuns, secondRuns)
	}
}

func TestRepeatedSubscriptionsCanUnsubscribeIndependently(t *testing.T) {
	scheduler := NewScheduler()
	count := NewWithScheduler(0, scheduler)

	first := 0
	second := 0
	unsubFirst := count.Subscribe(func() { first++ })
	count.Subscribe(func() { second++ })

	count.Set(1)
	scheduler.Flush()
	if first != 1 || second != 1 {
		t.Fatalf("initial subscriptions first=%d second=%d", first, second)
	}

	unsubFirst()
	count.Set(2)
	scheduler.Flush()
	if first != 1 || second != 2 {
		t.Fatalf("after unsubscribe first=%d second=%d", first, second)
	}
}

func TestNestedEffectsTrackTheirOwnDependencies(t *testing.T) {
	scheduler := NewScheduler()
	outerSignal := NewWithScheduler(0, scheduler)
	innerSignal := NewWithScheduler(10, scheduler)

	outerRuns := 0
	innerRuns := 0
	var inner *Effect
	NewEffectWithScheduler(func() {
		outerRuns++
		_ = outerSignal.Get()
		if inner == nil {
			inner = NewEffectWithScheduler(func() {
				innerRuns++
				_ = innerSignal.Get()
			}, scheduler)
		}
	}, scheduler)

	innerSignal.Set(11)
	scheduler.Flush()
	if outerRuns != 1 || innerRuns != 2 {
		t.Fatalf("inner update outerRuns=%d innerRuns=%d", outerRuns, innerRuns)
	}

	outerSignal.Set(1)
	scheduler.Flush()
	if outerRuns != 2 || innerRuns != 2 {
		t.Fatalf("outer update outerRuns=%d innerRuns=%d", outerRuns, innerRuns)
	}
}

func TestEffectStopUnsubscribes(t *testing.T) {
	scheduler := NewScheduler()
	count := NewWithScheduler(0, scheduler)

	runs := 0
	effect := NewEffectWithScheduler(func() {
		runs++
		_ = count.Get()
	}, scheduler)
	effect.Stop()
	count.Set(1)
	scheduler.Flush()
	if runs != 1 {
		t.Fatalf("effect ran after stop: %d", runs)
	}
}
