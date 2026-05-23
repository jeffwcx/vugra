package reactivity

import "reflect"

type Scheduler struct {
	queue   []func()
	queued  map[any]struct{}
	flushed bool
}

var DefaultScheduler = NewScheduler()

func NewScheduler() *Scheduler {
	return &Scheduler{
		queued: map[any]struct{}{},
	}
}

func (s *Scheduler) Schedule(fn func()) {
	if fn == nil {
		return
	}
	s.ScheduleKey(reflect.ValueOf(fn).Pointer(), fn)
}

func (s *Scheduler) ScheduleKey(key any, fn func()) {
	if fn == nil {
		return
	}
	if _, ok := s.queued[key]; ok {
		return
	}
	s.queued[key] = struct{}{}
	s.queue = append(s.queue, fn)
}

func (s *Scheduler) Flush() {
	for len(s.queue) > 0 {
		queue := s.queue
		s.queue = nil
		s.queued = map[any]struct{}{}
		for _, fn := range queue {
			fn()
		}
	}
	s.flushed = true
}

func (s *Scheduler) Pending() int {
	return len(s.queue)
}
