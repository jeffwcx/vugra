//go:build !darwin || !cgo

package csslayout

import "fmt"

func (e Engine) computeFFI(payload []byte) (Output, error) {
	return Output{}, fmt.Errorf("csslayout ffi unavailable on this platform")
}
