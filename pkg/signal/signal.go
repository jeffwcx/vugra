package signal

import "github.com/vugra/vugra/internal/reactivity"

type Signal[T any] struct {
	value *reactivity.Value[T]
}

func New[T any](initial T) Signal[T] {
	return Signal[T]{value: reactivity.New(initial)}
}

func (s *Signal[T]) Get() T {
	s.ensure()
	return s.value.Get()
}

func (s *Signal[T]) Set(value T) {
	s.ensure()
	s.value.Set(value)
}

func (s *Signal[T]) Update(update func(T) T) {
	s.ensure()
	s.value.Update(update)
}

func (s *Signal[T]) Subscribe(fn func()) func() {
	s.ensure()
	return s.value.Subscribe(fn)
}

func (s *Signal[T]) GetAny() any {
	return s.Get()
}

func (s *Signal[T]) SetAny(value any) {
	if typed, ok := value.(T); ok {
		s.Set(typed)
	}
}

func (s *Signal[T]) ensure() {
	if s.value == nil {
		var zero T
		s.value = reactivity.New(zero)
	}
}

type Int struct {
	value *reactivity.Value[int]
}

func NewInt(initial int) Int {
	return Int{value: reactivity.New(initial)}
}

func (s *Int) Get() int {
	s.ensure()
	return s.value.Get()
}

func (s *Int) Set(value int) {
	s.ensure()
	s.value.Set(value)
}

func (s *Int) Update(update func(int) int) {
	s.ensure()
	s.value.Update(update)
}

func (s *Int) Subscribe(fn func()) func() {
	s.ensure()
	return s.value.Subscribe(fn)
}

func (s *Int) GetAny() any {
	return s.Get()
}

func (s *Int) SetAny(value any) {
	if typed, ok := value.(int); ok {
		s.Set(typed)
	}
}

func (s *Int) ensure() {
	if s.value == nil {
		s.value = reactivity.New(0)
	}
}

type Bool struct {
	value *reactivity.Value[bool]
}

func NewBool(initial bool) Bool {
	return Bool{value: reactivity.New(initial)}
}

func (s *Bool) Get() bool {
	s.ensure()
	return s.value.Get()
}

func (s *Bool) Set(value bool) {
	s.ensure()
	s.value.Set(value)
}

func (s *Bool) Update(update func(bool) bool) {
	s.ensure()
	s.value.Update(update)
}

func (s *Bool) Subscribe(fn func()) func() {
	s.ensure()
	return s.value.Subscribe(fn)
}

func (s *Bool) GetAny() any {
	return s.Get()
}

func (s *Bool) SetAny(value any) {
	if typed, ok := value.(bool); ok {
		s.Set(typed)
	}
}

func (s *Bool) ensure() {
	if s.value == nil {
		s.value = reactivity.New(false)
	}
}

type String struct {
	value *reactivity.Value[string]
}

func NewString(initial string) String {
	return String{value: reactivity.New(initial)}
}

func (s *String) Get() string {
	s.ensure()
	return s.value.Get()
}

func (s *String) Set(value string) {
	s.ensure()
	s.value.Set(value)
}

func (s *String) Update(update func(string) string) {
	s.ensure()
	s.value.Update(update)
}

func (s *String) Subscribe(fn func()) func() {
	s.ensure()
	return s.value.Subscribe(fn)
}

func (s *String) GetAny() any {
	return s.Get()
}

func (s *String) SetAny(value any) {
	if typed, ok := value.(string); ok {
		s.Set(typed)
	}
}

func (s *String) ensure() {
	if s.value == nil {
		s.value = reactivity.New("")
	}
}
