package xorg

/*
#cgo linux CFLAGS: -I/usr/src -I/usr/local/include/
#cgo linux LDFLAGS: -L/usr/src -L/usr/local/lib -lX11 -lXtst -lXrandr -lxcb

#include "xorg.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

type XButtonCode int

const (
	XButtonCodeLeft   XButtonCode = 1
	XButtonCodeCenter XButtonCode = 2
	XButtonCodeRight  XButtonCode = 3
)

type ScreenSize struct {
	Width  int   `json:"width"`
	Height int   `json:"height"`
	Rate   int16 `json:"rate"`
}

type ScreenConfiguration struct {
	Width  int           `json:"width"`
	Height int           `json:"height"`
	Rates  map[int]int16 `json:"rates"`
}

var ScreenConfigurations = make(map[int]ScreenConfiguration)

var debounce_button = make(map[XButtonCode]time.Time)
var debounce_key = make(map[uint64]time.Time)
var mu = sync.Mutex{}

func init() {
	C.XGetScreenConfigurations()
}

func Display(display string) {
	mu.Lock()
	defer mu.Unlock()

	displayUnsafe := C.CString(display)
	defer C.free(unsafe.Pointer(displayUnsafe))

	C.XDisplaySet(displayUnsafe)
}

func Move(x, y int) {
	mu.Lock()
	defer mu.Unlock()

	C.XMove(C.int(x), C.int(y))
}

func Scroll(x, y int) {
	mu.Lock()
	defer mu.Unlock()

	C.XScroll(C.int(x), C.int(y))
}

func ButtonDown(code XButtonCode) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_button[code]; ok {
		return fmt.Errorf("debounced button %v", code)
	}

	debounce_button[code] = time.Now()

	C.XButton(C.uint(code), C.int(1))
	return nil
}

func KeyDown(code uint64) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_key[code]; ok {
		return fmt.Errorf("debounced key %v", code)
	}

	debounce_key[code] = time.Now()

	C.XKey(C.ulong(code), C.int(1))
	return nil
}

func ButtonUp(code XButtonCode) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_button[code]; !ok {
		return fmt.Errorf("debounced button %v", code)
	}

	delete(debounce_button, code)

	C.XButton(C.uint(code), C.int(0))
	return nil
}

func KeyUp(code uint64) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_key[code]; !ok {
		return fmt.Errorf("debounced key %v", code)
	}

	delete(debounce_key, code)

	C.XKey(C.ulong(code), C.int(0))
	return nil
}

func ResetKeys() {
	for code := range debounce_button {
		ButtonUp(code)

		delete(debounce_button, code)
	}
	for code := range debounce_key {
		KeyUp(code)

		delete(debounce_key, code)
	}
}

func CheckKeys(duration time.Duration) {
	t := time.Now()
	for code, start := range debounce_button {
		if t.Sub(start) < duration {
			continue
		}
		ButtonUp(code)

		delete(debounce_button, code)
	}
	for code, start := range debounce_key {
		if t.Sub(start) < duration {
			continue
		}
		KeyUp(code)

		delete(debounce_key, code)
	}
}

func ValidScreenSize(width int, height int, rate int) bool {
	for _, size := range ScreenConfigurations {
		if size.Width == width && size.Height == height {
			for _, fps := range size.Rates {
				if int16(rate) == fps {
					return true
				}
			}
		}
	}

	return false
}

func ChangeScreenSize(width int, height int, rate int) error {
	mu.Lock()
	defer mu.Unlock()

	for index, size := range ScreenConfigurations {
		if size.Width == width && size.Height == height {
			for _, fps := range size.Rates {
				if int16(rate) == fps {
					C.XSetScreenConfiguration(C.int(index), C.short(fps))
					return nil
				}
			}
		}
	}

	return fmt.Errorf("unknown configuration")
}

func GetScreenSize() *ScreenSize {
	mu.Lock()
	defer mu.Unlock()

	index := int(C.XGetScreenSize())
	rate := int16(C.XGetScreenRate())

	if conf, ok := ScreenConfigurations[index]; ok {
		return &ScreenSize{
			Width:  conf.Width,
			Height: conf.Height,
			Rate:   rate,
		}
	}

	return nil
}

//export goCreateScreenSize
func goCreateScreenSize(index C.int, width C.int, height C.int, mwidth C.int, mheight C.int) {
	ScreenConfigurations[int(index)] = ScreenConfiguration{
		Width:  int(width),
		Height: int(height),
		Rates:  make(map[int]int16),
	}
}

//export goSetScreenRates
func goSetScreenRates(index C.int, rate_index C.int, rate C.short) {
	ScreenConfigurations[int(index)].Rates[int(rate_index)] = int16(rate)
}