package g

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xllm-go/g/env"
	"github.com/xllm-go/g/internal"
	v1 "github.com/xllm-go/g/internal/v1"
)

type interfaces struct {
	//
}

func Sdk() interface {
	Env() *env.Environ
	Support(...string) *builder
	OnInitialized(func(), ...int)
	OnExited(func())
	OnError(fiber.Ctx, func(err interface{}))
} {
	return &interfaces{}
}

func (interfaces) Support(mod ...string) *builder {
	return (&builder{}).model(mod...)
}

func (interfaces) Env() *env.Environ {
	return env.Env
}

func (interfaces) OnInitialized(f func(), p ...int) {
	env.AddInitialized(f, p...)
}

func (interfaces) OnExited(f func()) {
	env.AddExited(f)
}

func (interfaces) OnError(ctx fiber.Ctx, f func(err interface{})) {
	v1.OnError(ctx, f)
}

func Execute() {
	internal.Execute()
}
