package model

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/xllm-go/g/logger"
)

const (
	Matchers    = "matchers"
	ThinkReason = "think_reason"
	ToolCall    = "tool_call"
)

type Ctx struct {
	ctx fiber.Ctx
	Record[string, any]

	Token string
	Type  string
}

func New(ctx fiber.Ctx) *Ctx {
	return &Ctx{
		ctx:    ctx,
		Record: make(Record[string, any]),

		Token: token(ctx),
	}
}

func (ctx *Ctx) Ctx() fiber.Ctx {
	return ctx.ctx
}

func (ctx *Ctx) Context() context.Context {
	cctx, ok := ctx.Get("context").(context.Context)
	if ok {
		return cctx
	}
	return context.Background()
}

func (ctx *Ctx) Cancel() {
	cancel, ok := ctx.Get("cancel").(context.CancelFunc)
	if ok {
		cancel()
	}
}

func (ctx *Ctx) StreamWriter(yield func(w func(*ChunkBodies) error), unix int64) {
	ctx.ctx.Set("content-type", "text/event-stream")
	ctx.ctx.Set("cache-control", "no-cache")
	ctx.ctx.Set("x-accel-buffering", "no")
	ctx.ctx.Set("x-accept-encoding", "gzip, deflate, br")
	ctx.ctx.Set("connection", "keep-alive")
	ctx.ctx.Set("transfer-encoding", "chunked")
	_ = ctx.ctx.SendStreamWriter(func(w *bufio.Writer) {
		yield(func(bodies *ChunkBodies) error {
			return write(w, bodies, unix)
		})
	})
	return
}

func (ctx *Ctx) Writer(msg interface{}) error {
	return ctx.ctx.JSON(msg)
}

func token(ctx fiber.Ctx) (token string) {
	token = ctx.Get("X-Api-Key")
	if token == "" {
		token = strings.TrimPrefix(ctx.Get("Authorization"), "Bearer ")
	}
	return
}

func write(w *bufio.Writer, bodies *ChunkBodies, unix int64) error {
	event := "data"
	var data string

	switch bodies.expr() {
	case -1:
		if bodies.Err == io.EOF {
			data = "[DONE]"
		} else {
			event = "error"
			resp := CreateResponse(bodies, unix)
			chunk, _ := json.Marshal(resp)
			data = string(chunk)
		}

	default:
		data = CreateResponse(bodies, unix).String()
	}

	err := sse(w, event, data)
	if err != nil {
		return err
	}

	if bodies.expr() == 0 {
		data = createStopResponse("tool_calls", unix).String()
		err = sse(w, event, data)
		if err != nil {
			return err
		}
	}

	return flush(w)
}

func sse(w *bufio.Writer, event, data string) (err error) {
	_, err = fmt.Fprintf(w, "%s: %s\n\n", event, data)
	if err != nil {
		logger.Sugar().Errorf("write sse data error: %v", err)
	}
	return
}

func flush(w *bufio.Writer) error {
	if err := w.Flush(); err != nil {
		logger.Sugar().Errorf("write sse data error: %v", err)
		return err
	}
	return nil
}
