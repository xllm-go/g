package g

import (
	"github.com/xllm-go/g/model"

	v1 "github.com/xllm-go/g/internal/v1"
)

type builder struct {
	slice []string
	yield func(ctx *model.Ctx) error
}

func (receiver *builder) model(mod ...string) *builder {
	receiver.slice = append(receiver.slice, mod...)
	return receiver
}

// 上下文对话
func (receiver *builder) Relay(yield func(ctx *model.Ctx) error) {
	receiver.yield = yield
	receiver.build("relay")
}

// 向量查询
func (receiver *builder) Embed(yield func(ctx *model.Ctx) error) {
	receiver.yield = yield
	receiver.build("embed")
}

// 文生图
func (receiver *builder) Image(yield func(ctx *model.Ctx) error) {
	receiver.yield = yield
	receiver.build("image")
}

func (receiver *builder) build(typed string) {
	v1.Put(typed, receiver.slice, receiver.yield)
}
