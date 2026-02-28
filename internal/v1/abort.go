package v1

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
)

type abortCtx struct {
	parent context.Context
	done   chan struct{}
	err    error
}

// 创建一个监听客户端断开的 Context
func newAbort(ctx fiber.Ctx) (context.Context, context.CancelFunc) {
	cctx := &abortCtx{
		parent: ctx.Context(),
		done:   make(chan struct{}),
	}

	loop := true
	cancel := func() {
		loop = false
		select {
		case <-cctx.done:
			// 已经关闭
		default:
			cctx.err = context.Canceled
			close(cctx.done)
		}
	}

	ch := make(chan byte, 1)
	conn := ctx.RequestCtx().Conn()
	go func() {
		for loop {
			if isClosed(conn) {
				ch <- 1
				loop = false
			}
			time.Sleep(time.Millisecond * 500)
		}
	}()

	// 后台轮询 IsAbandoned
	go func() {
		select {
		case <-cctx.parent.Done():
			select {
			case <-cctx.done:
			default:
				cctx.err = cctx.parent.Err()
				close(cctx.done)
			}
			return

		case <-ch:
			select {
			case <-cctx.done:
			default:
				cctx.err = context.Canceled
				close(cctx.done)
			}
			return

		case <-cctx.done:
			return
		}
	}()

	return cctx, cancel
}

func (c *abortCtx) Deadline() (time.Time, bool) {
	return c.parent.Deadline()
}

func (c *abortCtx) Done() <-chan struct{} {
	return c.done
}

func (c *abortCtx) Err() error {
	select {
	case <-c.done:
		return c.err
	default:
		return nil
	}
}

func (c *abortCtx) Value(key any) any {
	return c.parent.Value(key)
}

func isClosed(conn net.Conn) bool {
	// 获取底层文件描述符
	rawConn, ok := conn.(syscall.Conn)
	if !ok {
		return false
	}

	var closed bool
	rc, err := rawConn.SyscallConn()
	if err != nil {
		return false
	}

	_ = rc.Read(func(fd uintptr) bool {
		buf := make([]byte, 1)

		// 使用 MSG_PEEK：查看数据但不消费
		n, _, ioerr := syscall.Recvfrom(int(fd), buf, syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
		if n == 0 && ioerr == nil {
			// EOF：对端关闭
			closed = true
		}

		if errors.Is(ioerr, syscall.ECONNRESET) {
			closed = true
		}

		// EAGAIN/EWOULDBLOCK = 没有数据但连接正常
		return true
	})

	return closed
}
