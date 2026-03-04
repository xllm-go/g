package model

import (
	"strings"

	"github.com/xllm-go/g/env"
	"github.com/xllm-go/g/logger"

	regexp "github.com/dlclark/regexp2"
)

const (
	MatDefault  int = iota // 执行下一个匹配器
	MatMatching            // 匹配中, 字符被缓存
	MatMatched             // 匹配器命中，不再执行下一个
)

var (
	globalMatchers func(ctx *Ctx) []Matcher
)

// 匹配器接口
type Matcher interface {
	Match(content string, over bool) (state int, result string)
}

type obj struct {
	Match       string `mapstructure:"match"`
	Over        string `mapstructure:"over"`
	Notice      string `mapstructure:"notice"`
	Regex       string `mapstructure:"regex"`
	ThinkReason bool   `mapstructure:"think_reason"`
	Max         int    `mapstructure:"max"`
}

// 字符块匹配器，只向后匹配
type symbolMatcher struct {
	cache string // 缓存的字符
	Find  string // 字符块匹配前置，'*'则匹配任意
	// 具体的匹配实现, cache 仅在 MatMatched 状态有效
	H func(index int, content string) (state int, cache, result string)
}

func init() {
	env.AddInitialized(func() {
		var objs []obj
		err := env.Env.UnmarshalKey("matcher", &objs)
		if err != nil {
			logger.Sugar().Fatal(err)
		}
		if len(objs) != 0 {
			initMatchers(objs)
		}
	})
}

func initMatchers(objs []obj) {
	if len(objs) == 0 {
		return
	}

	globalMatchers = func(ctx *Ctx) (matchers []Matcher) {
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

			compile := regexp.MustCompile(`"(.+)" *: *"(.*)"`, regexp.ECMAScript)
			matched, err := compile.FindStringMatch(o.Regex)
			if err != nil {
				logger.Sugar().Errorf("the format has not been written correctly: matcher[%d].regex ==> %v", i, err)
				continue
			}

			regex, replacement := matched.GroupByNumber(1).String(), matched.GroupByNumber(2).String()
			c := regexp.MustCompile(regex, regexp.ECMAScript)
			var matcher *symbolMatcher

			matcher = &symbolMatcher{
				Find: match,
				H: func(index int, content string) (state int, cache, result string) {

					if over != "" {
						if !strings.Contains(content, over) {
							return MatMatching, "", content
						}
						idx := strings.LastIndex(content, over)
						cache = content[idx+len(over):]
						content = content[:idx+len(over)]
					} else {
						r := []rune(content)
						if index+maxLen > len(r)-1 {
							return MatMatching, "", content
						}
					}

					logger.Sugar().Infof("execute matcher[%s] content:\n%s", matcher.Find, content)
					result, err = c.Replace(content, replacement, 0, 1)
					if o.ThinkReason && content != "" {
						ctx.Put(ThinkReason, result)
						return MatMatched, cache, ""
					}

					if err != nil {
						logger.Sugar().Warn("compile failed: "+regex, err)
						return MatMatched, cache, content
					}
					return MatMatched, cache, result
				},
			}
			matchers = append(matchers, matcher)
		}
		return
	}
}

func NewMatchers(ctx *Ctx) (slice []Matcher) {
	slice = make([]Matcher, 0)
	if globalMatchers != nil {
		slice = append(slice, globalMatchers(ctx)...)
	}

	// TOOL CALL 匹配器
	over := "</tool_call>"
	slice = append(slice, &symbolMatcher{
		Find: "<tool_call>",
		H: func(index int, content string) (state int, cache, result string) {
			if !strings.Contains(content, over) {
				return MatMatching, "", content
			}
			idx := strings.LastIndex(content, over)
			cache = content[idx+len(over):]
			content = content[:idx+len(over)]

			logger.Sugar().Infof("execute matcher[<tool_call>] content:\n%s", content)

			// 处理标签
			content = strings.TrimSpace(content)
			ctx.Put(ToolCall, content[11:len(content)-12])
			return MatMatched, cache, ""
		},
	})

	// "<|im_end|>"
	slice = append(slice, &symbolMatcher{
		Find: "<|im_end|>",
		H: func(index int, content string) (state int, cache, result string) {
			state = MatMatched
			ctx.Clone()
			return
		},
	})

	return
}

func NewMatcher(find string, h func(index int, content string) (state int, cache, result string)) Matcher {
	return &symbolMatcher{
		Find: find,
		H:    h,
	}
}

// MAT_DEFAULT	没有命中，继续执行下一个。
// MAT_MATCHING 匹配中，缓存消息不执行下一个。
// MAT_MATCHED 	命中，不再执行下一个。
func ExecMatchers(ctx *Ctx, raw string, done bool) string {
	matchers := JustValue[string, []Matcher](ctx.Record, Matchers)
	s := MatDefault
	for _, mat := range matchers {
		s, raw = mat.Match(raw, done)
		if s == MatDefault {
			continue
		}
		break
	}
	return raw
}

func (mat *symbolMatcher) Match(content string, over bool) (state int, result string) {
	content = mat.cache + content
	state = MatDefault
	// MatDefault 没有命中
	// MatMatching 匹配中
	// MatMatched 命中了
	var (
		index = 0
		find  = []rune(mat.Find)
		rc    = []rune(content)

		pos = 0
		idx = -1
	)

	if mat.Find == "" {
		state = MatMatched
		goto state
	}

	for index = range rc {
		var ch rune
		if len(find) == pos {
			// 到这里就代表命中了，检查一下
			if strings.HasSuffix(content, string(find)) {
				state = MatMatched
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
			state = MatDefault
			continue
		}

		if idx == -1 || idx == index-1 {
			pos++
			idx = index
			state = MatMatching
			continue
		}
	}

state:
	// 没有命中，返回所有内容（包括cache）
	if state == MatDefault {
		mat.cache = ""
		result = content
		return
	}

	// 还在匹配中，再次校验是否命中
	if state == MatMatching {
		mat.cache = content // 缓存
		if strings.Contains(content, mat.Find) {
			state = MatMatched // 命中
		} else {
			result = "" // 等待下次输入
			return
		}
	}

	if mat.H != nil {
		var leaveCache string
		state, leaveCache, result = mat.H(index, content) // 执行下游自定义处理
		if state == MatMatched {                          // 处理完毕
			mat.cache = leaveCache
			return
		}
		if state == MatMatching { // 还在处理中
			if over { // 已经没有后续输入了
				return MatDefault, content
			}
			mat.cache = result
			return state, ""
		}

		return state, content
	} else {
		result = content
		mat.cache = ""
	}

	return
}
