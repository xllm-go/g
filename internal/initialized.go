package internal

import (
	"os"
	"os/signal"
	"syscall"

	v1 "github.com/xllm-go/g/internal/v1"
)

var (
	inits  = make([]func(), 0)
	exits  = make([]func(), 0)
	panics = make([]func(interface{}), 0)
)

func init() {
	v1.SetPanic(func(err interface{}) {
		for _, yield := range panics {
			yield(err)
		}
	})
}

func AddPanic(apply func(interface{})) { panics = append(panics, apply) }
func AddInitialized(apply func())      { inits = append(inits, apply) }
func AddExited(apply func())           { exits = append(exits, apply) }

func Initialized() {
	for _, yield := range inits {
		yield()
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func(ch chan os.Signal) {
		<-ch
		for _, yield := range exits {
			yield()
		}
		os.Exit(0)
	}(osSignal)
}
