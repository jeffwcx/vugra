package reactivity

type dependency interface {
	subscribeEffect(*Effect) func()
}

type Effect struct {
	fn        func()
	scheduler *Scheduler
	cleanups  []func()
	deps      map[dependency]struct{}
	running   bool
}

func NewEffect(fn func()) *Effect {
	return NewEffectWithScheduler(fn, DefaultScheduler)
}

func NewEffectWithScheduler(fn func(), scheduler *Scheduler) *Effect {
	if scheduler == nil {
		scheduler = DefaultScheduler
	}
	e := &Effect{
		fn:        fn,
		scheduler: scheduler,
		deps:      map[dependency]struct{}{},
	}
	e.Run()
	return e
}

func (e *Effect) Run() {
	if e.running || e.fn == nil {
		return
	}
	e.cleanup()
	e.running = true
	pushEffect(e)
	e.fn()
	popEffect(e)
	e.running = false
}

func (e *Effect) Stop() {
	e.cleanup()
}

func (e *Effect) schedule() {
	e.scheduler.ScheduleKey(e, e.Run)
}

func (e *Effect) track(dep dependency) {
	if dep == nil {
		return
	}
	if _, ok := e.deps[dep]; ok {
		return
	}
	e.deps[dep] = struct{}{}
	e.cleanups = append(e.cleanups, dep.subscribeEffect(e))
}

func (e *Effect) cleanup() {
	for _, cleanup := range e.cleanups {
		cleanup()
	}
	e.cleanups = nil
	e.deps = map[dependency]struct{}{}
}

var effectStack []*Effect

func currentEffect() *Effect {
	if len(effectStack) == 0 {
		return nil
	}
	return effectStack[len(effectStack)-1]
}

func pushEffect(effect *Effect) {
	effectStack = append(effectStack, effect)
}

func popEffect(effect *Effect) {
	if len(effectStack) == 0 {
		return
	}
	effectStack = effectStack[:len(effectStack)-1]
}
