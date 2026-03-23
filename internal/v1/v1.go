package v1

import (
	"container/list"
	"crypto/tls"
	"fmt"
	"iter"
	"maps"
	"time"

	"github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/xllm-go/g/interceptor"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

var (
	con    = &container{}
	panics = make(map[fiber.Ctx]func(err interface{}))
)

func OnError(ctx fiber.Ctx, f func(err interface{})) {
	panics[ctx] = f
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
		if con.rm == nil {
			con.rm = make(map[*list.List]func(ctx *model.Ctx) error)
		}
		con.rm[ctr] = f
	case "image":
		if con.im == nil {
			con.im = make(map[*list.List]func(ctx *model.Ctx) error)
		}
		con.im[ctr] = f
	}
}

// 模型迭代器
func Models() iter.Seq[model.Model] {
	return func(yield func(model.Model) bool) {
		keys := maps.Keys(con.rm)
		for ctr := range keys {
			for curr := ctr.Front(); curr != nil; curr = curr.Next() {
				yield(model.Model{
					Object:  "model",
					Id:      curr.Value.(string),
					By:      "chatgpt-adapter:v3.0.1",
					Created: time.Now().Unix(),
				})
			}
		}

		keys = maps.Keys(con.im)
		for ctr := range keys {
			for curr := ctr.Front(); curr != nil; curr = curr.Next() {
				yield(model.Model{
					Object:  "model",
					Id:      curr.Value.(string),
					By:      "chatgpt-adapter:v3.0.1",
					Created: time.Now().Unix(),
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
			logger.Sugar().Errorf("panic: %v", err)
			if yield, ok := panics[ctx]; ok {
				delete(panics, ctx)
				yield(err)
			}
		},
	}))

	app.Use(zap.New(zap.Config{
		Logger: logger.Logger(),
	}))

	app.Use(func(ctx fiber.Ctx) (err error) {
		logger.Sugar().Infof("-------------------- NEW START --------------------")
		err = ctx.Next()
		delete(panics, ctx)
		return
	})

	app.Get("/", index)

	app.Post("v1/chat/completions", completions)
	app.Post("v1/object/completions", completions)
	app.Post("proxies/v1/chat/completions", completions)

	app.Post("v1/images/generations", generations)
	app.Post("v1/object/generations", generations)
	app.Post("proxies/v1/images/generations", generations)

	app.Get("v1/models", models)
	app.Get("proxies/v1/models", models)

	err := app.Listen(addr, fiber.ListenConfig{
		TLSMinVersion:      tls.VersionTLS12,
		ListenerNetwork:    fiber.NetworkTCP,
		ShutdownTimeout:    10 * time.Second,
		UnixSocketFileMode: 0o770,
	})
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

func models(ctx fiber.Ctx) (err error) {
	data := make([]model.Model, 0)
	for mod := range Models() {
		data = append(data, mod)
	}
	return ctx.JSON(map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

func completions(ctx fiber.Ctx) (err error) {
	completion := new(model.Completion)
	if err = ctx.Bind().JSON(completion); err != nil {
		return
	}

	cctx := model.New(ctx)
	cctx.Type = "relay"
	cctx.Put("completion", completion)
	abort, cancel := newAbort(ctx)
	cctx.Put("cancel", cancel)
	cctx.Put("context", abort)

	if con.Support(cctx, completion.Model) {
		cctx.Put(interceptor.Matcher, interceptor.NewInterceptors(cctx))
		return con.Relay(cctx)
	}

	return writeError(ctx, fmt.Sprintf("model [%s] is not found", completion.Model))
}

func generations(ctx fiber.Ctx) (err error) {
	generation := new(model.Generation)
	if err = ctx.Bind().JSON(generation); err != nil {
		return
	}

	cctx := model.New(ctx)
	cctx.Type = "image"
	cctx.Put("generation", generation)
	if con.Support(cctx, generation.Model) {
		return con.Relay(cctx)
	}

	return writeError(ctx, fmt.Sprintf("model [%s] is not found", generation.Model))
}

func writeError(ctx fiber.Ctx, msg string) (err error) {
	return ctx.Status(fiber.StatusInternalServerError).
		JSON(model.Record[string, any]{
			"error": msg,
		})
}
