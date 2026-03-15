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

// 字符块匹配器，只向后匹配
type symbolInterceptor struct {
	cache string // 缓存的字符
	Find  string // 字符块匹配前置，'*'则匹配任意
	// 具体的匹配实现, cache 仅在 Matched 状态有效
	H func(index int, content string) (state int, cache, result string)
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
			maxLen := o.Max
			if maxLen == 0 {
				maxLen = 5
			}

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

			var matcher Interceptor = &symbolInterceptor{
				Find: match,
				H: func(index int, content string) (state int, cache, result string) {

					if over != "" {
						if !strings.Contains(content, over) {
							return Matching, "", content
						}
						idx := strings.LastIndex(content, over)
						cache = content[idx+len(over):]
						content = content[:idx+len(over)]
					} else {
						r := []rune(content)
						if index+maxLen > len(r)-1 {
							return Matching, "", content
						}
					}

					logger.Sugar().Infof("execute matcher[%s] content:\n%s", match, content)
					result, err := c.Replace(content, replacement, 0, 1)
					if o.ThinkReason && content != "" {
						ctx.Put(ThinkReason, result)
						return Matched, cache, ""
					}

					if err != nil {
						logger.Sugar().Warn("compile failed: "+regex, err)
						return Matched, cache, content
					}
					return Matched, cache, result
				},
			}
			matchers = append(matchers, matcher)
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
	over := "</tool_call>"
	slice = append(slice, &symbolInterceptor{
		Find: "<tool_call>",
		H: func(index int, content string) (state int, cache, result string) {
			if !strings.Contains(content, over) {
				return Matching, "", content
			}
			idx := strings.LastIndex(content, over)
			cache = content[idx+len(over):]
			content = content[:idx+len(over)]

			logger.Sugar().Infof("execute matcher[<tool_call>] content:\n%s", content)

			// 处理标签
			content = strings.TrimSpace(content)
			ctx.Put(ToolCall, content[11:len(content)-12])
			return Matched, cache, ""
		},
	})

	// "<|im_end|>"
	slice = append(slice, &symbolInterceptor{
		Find: "<|im_end|>",
		H: func(index int, content string) (state int, cache, result string) {
			state = Matched
			ctx.Clone()
			return
		},
	})

	return
}

func AddInterceptor(ctx *model.Ctx, in ...Interceptor) {
	interceptors := model.JustValue[string, []Interceptor](ctx.Record, Matcher)
	if interceptors == nil {
		return
	}

	for _, i := range in {
		interceptors = append(interceptors, i.(Interceptor))
	}

	ctx.Put(Matcher, interceptors)
}

// MAT_DEFAULT	没有命中，继续执行下一个。
// MAT_MATCHING 匹配中，缓存消息不执行下一个。
// MAT_MATCHED 	命中，不再执行下一个。
func ExecuteInterceptors(ctx *model.Ctx, raw string, done bool) string {
	matchers := model.JustValue[string, []Interceptor](ctx.Record, Matcher)
	s := Default
	for _, mat := range matchers {
		s, raw = mat.scan(raw, done)
		if s == Default {
			continue
		}
		break
	}
	return raw
}

func (mat *symbolInterceptor) scan(content string, over bool) (state int, result string) {
	content = mat.cache + content
	state = Default
	// Default 没有命中
	// Matching 匹配中
	// Matched 命中了
	var (
		index = 0
		find  = []rune(mat.Find)
		rc    = []rune(content)

		pos = 0
		idx = -1
	)

	if mat.Find == "" {
		state = Matched
		goto state
	}

	for index = range rc {
		var ch rune
		if len(find) == pos {
			// 到这里就代表命中了，检查一下
			if strings.HasSuffix(content, string(find)) {
				state = Matched
			}
			if mat.H != nil {
				break
			}
			continue
		}

		ch = find[pos]
		if ch != rc[index] {
			pos = 0
			idx = -1
			state = Default
			continue
		}

		if idx == -1 || idx == index-1 {
			pos++
			idx = index
			state = Matching
			continue
		}
	}

state:
	// 没有命中，返回所有内容（包括cache）
	if state == Default {
		mat.cache = ""
		result = content
		return
	}

	// 还在匹配中，再次校验是否命中
	if state == Matching {
		mat.cache = content // 缓存
		if strings.Contains(content, mat.Find) {
			state = Matched // 命中
		} else {
			result = "" // 等待下次输入
			return
		}
	}

	if mat.H != nil {
		var leaveCache string
		state, leaveCache, result = mat.H(index, content) // 执行下游自定义处理
		if state == Matched {                             // 处理完毕
			mat.cache = leaveCache
			return
		}
		if state == Matching { // 还在处理中
			if over { // 已经没有后续输入了
				return Default, content
			}
			mat.cache = result
			return state, ""
		}

		return state, content
	}

	result = content
	mat.cache = ""
	return
}
