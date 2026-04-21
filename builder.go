package g

import (
	"time"

	"github.com/xllm-go/g/logger"
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

// 文生图
func (receiver *builder) Image(yield func(ctx *model.Ctx) error, retry ...int) {
	if l := len(retry); l > 0 {
		sleep := 0
		if l > 1 && retry[1] > 0 {
			sleep = retry[1]
		}

		ex := yield
		yield = func(ctx *model.Ctx) error {
			return execute(func() error {
				return ex(ctx)
			}, retry[0], sleep)
		}
	}
	receiver.yield = yield
	receiver.build("image")
}

func (receiver *builder) build(typed string) {
	v1.Put(typed, receiver.slice, receiver.yield)
}

func execute(yield func() error, retry int, sleep int) (err error) {
	if retry < 0 {
		retry = 0
	}

label:
	err = yield()
	if err == nil {
		return
	}

	if retry > 0 {
		retry--
		logger.Sugar().Warnf("retry by err: %v", err)
		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
		}
		goto label
	}

	return
}
