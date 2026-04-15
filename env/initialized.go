package env

import (
	"cmp"
	"os"
	"os/signal"
	"slices"
	"syscall"
)

var (
	inits = make([]initialized, 0)
	exits = make([]func(), 0)
)

type initialized struct {
	apply    func()
	priority int
}

func AddExited(apply func()) { exits = append(exits, apply) }
func AddInitialized(apply func(), priority ...int) {
	p := 10
	if len(priority) > 0 {
		p = priority[0]
	}

	inits = append(inits, initialized{apply, p})
	slices.SortFunc(inits, func(a, b initialized) int {
		return cmp.Compare(a.priority, b.priority)
	})
}

func Initialized() {

	for _, yield := range inits {
		yield.apply()
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
