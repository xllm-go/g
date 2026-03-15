package v1

import (
	"maps"

	"container/list"

	"github.com/xllm-go/g/model"
)

type container struct {
	rm map[*list.List]func(ctx *model.Ctx) error
	im map[*list.List]func(ctx *model.Ctx) error
}

func (_this *container) Support(ctx *model.Ctx, mod string) bool {
	var currMap map[*list.List]func(ctx *model.Ctx) error
	switch ctx.Type {
	case "relay":
		if _this.rm == nil {
			return false
		}
		currMap = _this.rm

	case "image":
		if _this.im == nil {
			return false
		}
		currMap = _this.im

	default:
		return false
	}

	keys := maps.Keys(currMap)
	for ctr := range keys {
		for curr := ctr.Front(); curr != nil; curr = curr.Next() {
			value := curr.Value.(string)
			if mod == value {
				ctx.Record.Put("relay", currMap[ctr])
				return true
			}
		}
	}

	return false
}

// 上下文对话
func (*container) Relay(ctx *model.Ctx) (err error) {
	relay, ok := model.GetValue[string, func(ctx *model.Ctx) error](ctx.Record, "relay")
	if !ok {
		return
	}
	return relay(ctx)
}
