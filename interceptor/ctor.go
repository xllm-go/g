package interceptor

import (
	"strings"

	"github.com/xllm-go/g/env"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"

	regexp "github.com/dlclark/regexp2"
)

const (
	Default  int = iota // 执行下一个匹配器
	Matching            // 匹配中, 字符被缓存
	Matched             // 匹配器命中，不再执行下一个

	Matcher     = "matchers"
	ThinkReason = "think_reason"
	ToolCall    = "tool_call"
)

var (
	interceptorConstructor func(ctx *model.Ctx) []Interceptor
)

// 匹配器接口
type Interceptor interface {
	scan(content string, over bool) (state int, result string)
}

type mapper struct {
	Match       string `mapstructure:"match"`
	Over        string `mapstructure:"over"`
	Notice      string `mapstructure:"notice"`
	Regex       string `mapstructure:"regex"`
	ThinkReason bool   `mapstructure:"think_reason"`
	Max         int    `mapstructure:"max"`
}

func init() {
	env.AddInitialized(func() {
		var objs []mapper
		err := env.Env.UnmarshalKey("matcher", &objs)
		if err != nil {
			logger.Sugar().Fatal(err)
		}
		if len(objs) != 0 {
			initInterceptor(objs)
		}
	})
}

func initInterceptor(objs []mapper) {
	if len(objs) == 0 {
		return
	}

	interceptorConstructor = func(ctx *model.Ctx) (matchers []Interceptor) {
		for i, o := range objs {
			match, over := o.Match, o.Over
			if o.Regex == "" {
				logger.Sugar().Errorf("no regular processing is configured: matcher[%d].regex", i)
				continue
			}

			var c *regexp.Regexp
			var regex, replacement string

			if o.Regex != "blank" {
				compile := regexp.MustCompile(`"(.+)" *: *"(.*)"`, regexp.ECMAScript)
				matched, err := compile.FindStringMatch(o.Regex)
				if err != nil {
					logger.Sugar().Errorf("the format has not been written correctly: matcher[%d].regex ==> %v", i, err)
					continue
				}

				regex, replacement = matched.GroupByNumber(1).String(), matched.GroupByNumber(2).String()
				c = regexp.MustCompile(regex, regexp.ECMAScript)
			}

			H := func(index int, content string) (state int, result string) {
				if o.Regex == "blank" {
					return Matched, ""
				}

				result, err := c.Replace(content, replacement, 0, 1)
				if o.ThinkReason && content != "" {
					ctx.Put(ThinkReason, result)
					return Matched, ""
				}

				if err != nil {
					logger.Sugar().Errorf("compile failed: %s\n         error: %v", regex, err)
					return Matched, content
				}
				return Matched, result
			}

			// 标记包裹拦截
			if over != "" {
				matchers = append(matchers, &markupInterceptor{
					Find: match,
					Over: o.Over,
					H:    H,
				})
				return
			}

			// 标记步长拦截
			matchers = append(matchers, &expressionInterceptor{
				Find: match,
				Max:  o.Max,
				H:    H,
			})

		}
		return
	}
}

func NewInterceptors(ctx *model.Ctx) (slice []Interceptor) {
	slice = make([]Interceptor, 0)
	if interceptorConstructor != nil {
		slice = append(slice, interceptorConstructor(ctx)...)
	}

	// TOOL CALL 匹配器
	slice = append(slice, &markupInterceptor{
		Find: "<tool_call>",
		Over: "</tool_call>",
		H: func(index int, content string) (state int, result string) {
			content = strings.TrimSpace(content)
			ctx.Put(ToolCall, content[11:len(content)-12])
			return Matched, ""
		},
	})

	// "<|im_end|>"
	slice = append(slice, &symbolInterceptor{
		Find: "<|im_end|>",
		H: func(_ int, content string) (state int, result string) {
			state = Matched
			ctx.Clone()
			return
		},
	})

	return
}

func AppendInterceptor(ctx *model.Ctx, in ...Interceptor) {
	interceptors := model.JustValue[string, []Interceptor](ctx.Record, Matcher)
	if interceptors == nil {
		return
	}

	for _, i := range in {
		interceptors = append(interceptors, i.(Interceptor))
	}

	ctx.Put(Matcher, interceptors)
}

func ExecuteInterceptors(ctx *model.Ctx, chunk string, over bool) string {
	matchers := model.JustValue[string, []Interceptor](ctx.Record, Matcher)
	state := Default
	for _, mat := range matchers {
		state, chunk = mat.scan(chunk, over)
		if state != Default {
			break
		}
	}
	return chunk
}
