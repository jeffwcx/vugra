package reactivity

// Package reactivity provides explicit Signal types, effects, and batched
// invalidation scheduling.

type Signal[T any] interface {
	Get() T
	Set(T)
	Update(func(T) T)
	Subscribe(func()) func()
}

type Value[T any] struct {
	value       T
	subscribers map[int]signalSubscriber
	nextID      int
	scheduler   *Scheduler
}

type signalSubscriber struct {
	key any
	fn  func()
}

type subscriptionKey struct {
	source any
	id     int
}

func New[T any](initial T) *Value[T] {
	return NewWithScheduler(initial, DefaultScheduler)
}

func NewWithScheduler[T any](initial T, scheduler *Scheduler) *Value[T] {
	if scheduler == nil {
		scheduler = DefaultScheduler
	}
	return &Value[T]{
		value:       initial,
		subscribers: map[int]signalSubscriber{},
		scheduler:   scheduler,
	}
}

func (s *Value[T]) Get() T {
	if current := currentEffect(); current != nil {
		current.track(s)
	}
	return s.value
}

func (s *Value[T]) Set(next T) {
	s.value = next
	for _, subscriber := range s.subscribers {
		s.scheduler.ScheduleKey(subscriber.key, subscriber.fn)
	}
}

func (s *Value[T]) Update(update func(T) T) {
	s.Set(update(s.value))
}

func (s *Value[T]) Subscribe(subscriber func()) func() {
	if subscriber == nil {
		return func() {}
	}
	id := s.nextID
	s.nextID++
	s.subscribers[id] = signalSubscriber{
		key: subscriptionKey{source: s, id: id},
		fn:  subscriber,
	}
	active := true
	return func() {
		if !active {
			return
		}
		active = false
		delete(s.subscribers, id)
	}
}

func (s *Value[T]) subscribeEffect(effect *Effect) func() {
	return s.Subscribe(effect.schedule)
}

type Int = Value[int]
type String = Value[string]
type Bool = Value[bool]
