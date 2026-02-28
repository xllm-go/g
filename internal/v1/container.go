package v1

import (
	"maps"

	"container/list"

	"github.com/xllm-go/g/model"
)

type container struct {
	relayMap map[*list.List]func(ctx *model.Ctx) error
	embedMap map[*list.List]func(ctx *model.Ctx) error
	imageMap map[*list.List]func(ctx *model.Ctx) error
}

func (_this *container) Support(ctx *model.Ctx, mod string) bool {
	var currMap map[*list.List]func(ctx *model.Ctx) error
	switch ctx.Type {
	case "relay":
		if _this.relayMap == nil {
			return false
		}
		currMap = _this.relayMap

	case "embed":
		if _this.embedMap == nil {
			return false
		}
		currMap = _this.embedMap

	case "image":
		if _this.imageMap == nil {
			return false
		}
		currMap = _this.imageMap

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
