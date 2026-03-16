package g

import (
	"github.com/xllm-go/g/env"
	"github.com/xllm-go/g/internal"
)

type interfaces struct {
	//
}

func Sdk() interface {
	Env() *env.Environ
	Support(...string) *builder
	OnInitialized(func())
	OnExited(func())
	OnPanic(func(interface{}))
} {
	return &interfaces{}
}

func (interfaces) Support(mod ...string) *builder {
	return (&builder{}).model(mod...)
}

func (interfaces) Env() *env.Environ {
	return env.Env
}

func (interfaces) OnInitialized(f func()) {
	env.AddInitialized(f)
}

func (interfaces) OnExited(f func()) {
	env.AddExited(f)
}

func (interfaces) OnPanic(f func(interface{})) {
	env.AddPanic(f)
}

func Execute() {
	internal.Execute()
}
