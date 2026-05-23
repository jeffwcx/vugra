package host_test

import (
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/host"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/template"
)

func TestMemoryHostRunsPointerEvents(t *testing.T) {
	templateDoc := template.Parse(`
<div class="counter">
  <p>{{ count }}</p>
  <button @click="Inc">+</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Count signal.Int `+"`vugra:\"count\"`"+`
}
func (s *State) Inc() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Counter", Template: templateDoc, Go: goMeta})
	sheet := style.Parse(`
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
}
`, style.BasePosition{})

	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	memory := &host.MemoryHost{
		Keys: []host.KeyEvent{{Key: "Tab"}, {Key: "Enter"}},
	}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() { count.Update(func(value int) int { return value + 1 }) },
		},
	}, memory, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{},
	})
	if err := memory.Run(app); err != nil {
		t.Fatalf("run host: %v", err)
	}
	if !frameContainsText(memory.LastFrame(), "1") {
		t.Fatalf("last frame did not update: %+v", memory.LastFrame())
	}
	tree := memory.LastAccessibilityTree()
	if len(tree) != 1 || len(tree[0].Children) == 0 {
		t.Fatalf("missing accessibility tree: %+v", tree)
	}
}

func TestMemoryHostRunsTextEvents(t *testing.T) {
	templateDoc := template.Parse(`
<div>
  <input :value="name">
  <p>{{ name }}</p>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Name signal.String `+"`vugra:\"name\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Input", Template: templateDoc, Go: goMeta})
	scheduler := reactivity.NewScheduler()
	name := runtime.NewSignal("", scheduler)
	memory := &host.MemoryHost{
		Clicks: []host.PointerEvent{{X: 1, Y: 1}},
		Text:   []host.TextEvent{{Text: "Grace"}},
	}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"name": name},
		Methods: map[string]func(){},
	}, memory, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{},
	})
	if err := memory.Run(app); err != nil {
		t.Fatalf("run host: %v", err)
	}
	if name.Get() != "Grace" {
		t.Fatalf("name = %q", name.Get())
	}
	if !frameContainsText(memory.LastFrame(), "Grace") {
		t.Fatalf("last frame did not update: %+v", memory.LastFrame())
	}
}

func frameContainsText(frame []renderer.Command, text string) bool {
	for _, command := range frame {
		if command.Kind == "text" && command.Text == text {
			return true
		}
	}
	return false
}
