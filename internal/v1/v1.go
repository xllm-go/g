package v1

import (
	"container/list"
	"fmt"
	"iter"
	"maps"
	"time"

	"github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

var (
	c         = &container{}
	panicFunc func(err interface{})
)

func SetPanic(f func(interface{})) {
	panicFunc = f
}

func Put(typed string, mod []string, f func(ctx *model.Ctx) error) {
	if len(mod) == 0 {
		return
	}

	ctr := list.New()
	for _, id := range mod {
		ctr.PushBack(id)
	}

	switch typed {
	case "relay":
		if c.relayMap == nil {
			c.relayMap = make(map[*list.List]func(ctx *model.Ctx) error)
		}
		c.relayMap[ctr] = f
	case "embed":
		if c.embedMap == nil {
			c.embedMap = make(map[*list.List]func(ctx *model.Ctx) error)
		}
		c.embedMap[ctr] = f
	case "image":
		if c.imageMap == nil {
			c.imageMap = make(map[*list.List]func(ctx *model.Ctx) error)
		}
		c.imageMap[ctr] = f
	}
}

// 模型迭代器
func Models() iter.Seq[model.Model] {
	return func(yield func(model.Model) bool) {
		keys := maps.Keys(c.relayMap)
		for ctr := range keys {
			for curr := ctr.Front(); curr != nil; curr = curr.Next() {
				yield(model.Model{
					Object:  "model",
					Id:      curr.Value.(string),
					By:      "chatgpt-adapter:v3.0.1",
					Created: time.Now().Second(),
				})
			}
		}

		keys = maps.Keys(c.embedMap)
		for ctr := range keys {
			for curr := ctr.Front(); curr != nil; curr = curr.Next() {
				yield(model.Model{
					Object:  "model",
					Id:      curr.Value.(string),
					By:      "chatgpt-adapter:v3.0.1",
					Created: time.Now().Second(),
				})
			}
		}

		keys = maps.Keys(c.imageMap)
		for ctr := range keys {
			for curr := ctr.Front(); curr != nil; curr = curr.Next() {
				yield(model.Model{
					Object:  "model",
					Id:      curr.Value.(string),
					By:      "chatgpt-adapter:v3.0.1",
					Created: time.Now().Second(),
				})
			}
		}
	}
}

// 初始化fiber api
func Initialized(addr string) {
	app := fiber.New(fiber.Config{
		BodyLimit:      20 * 1024 * 1024,
		ReadBufferSize: 127 * 1024,
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(ctx fiber.Ctx, err interface{}) {
			if panicFunc != nil {
				panicFunc(err)
			}
			logger.Sugar().Errorf("panic: %v", err)
		},
	}))

	app.Use(zap.New(zap.Config{
		Logger: logger.Logger(),
	}))

	app.Use(func(ctx fiber.Ctx) (err error) {
		abort, _ := newAbort(ctx)
		ctx.SetContext(abort)
		//defer cancel()
		return ctx.Next()
	})

	app.Get("/", index)

	app.Post("v1/chat/completions", completions)
	app.Post("v1/object/completions", completions)
	app.Post("proxies/v1/chat/completions", completions)

	app.Post("/v1/embeddings", embeddings)
	app.Post("proxies/v1/embeddings", embeddings)

	app.Post("v1/images/generations", generations)
	app.Post("v1/object/generations", generations)
	app.Post("proxies/v1/images/generations", generations)

	err := app.Listen(addr)
	if err != nil {
		panic(err)
	}
}

func index(ctx fiber.Ctx) error {
	ctx.Set("content-type", "text/html")
	return JustError(
		ctx.WriteString("<div style='color:green'>success ~</div>"),
	)
}

func completions(ctx fiber.Ctx) (err error) {
	completion := new(model.Completion)
	if err = ctx.Bind().JSON(completion); err != nil {
		return
	}

	cctx := model.New(ctx)
	cctx.Type = "relay"
	cctx.Put("completion", completion)
	if c.Support(cctx, completion.Model) {
		return c.Relay(cctx)
	}

	return writeError(ctx, fmt.Sprintf("model [%s] is not found", completion.Model))
}

func embeddings(ctx fiber.Ctx) (err error) {
	embedding := new(model.Embedding)
	if err = ctx.Bind().JSON(embedding); err != nil {
		return
	}

	cctx := model.New(ctx)
	cctx.Type = "embed"
	cctx.Put("embedding", embedding)
	if c.Support(cctx, embedding.Model) {
		return c.Relay(cctx)
	}

	return writeError(ctx, fmt.Sprintf("model [%s] is not found", embedding.Model))
}

func generations(ctx fiber.Ctx) (err error) {
	generation := new(model.Generation)
	if err = ctx.Bind().JSON(generation); err != nil {
		return
	}

	cctx := model.New(ctx)
	cctx.Type = "image"
	cctx.Put("generation", generation)
	if c.Support(cctx, generation.Model) {
		return c.Relay(cctx)
	}

	return writeError(ctx, fmt.Sprintf("model [%s] is not found", generation.Model))
}

func writeError(ctx fiber.Ctx, msg string) (err error) {
	return ctx.Status(fiber.StatusInternalServerError).
		JSON(model.Record[string, any]{
			"error": msg,
		})
}
