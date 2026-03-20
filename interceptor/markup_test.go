package interceptor

import (
	"strings"
	"sync"
	"testing"

	regexp "github.com/dlclark/regexp2"
	"github.com/xllm-go/g/logger"
)

func TestThinkReason(t *testing.T) {
	logger.InitLogger(
		"log",
		logger.DebugLevel,
	)

	chunk := []string{
		"\n\n<think>\n",
		"用户只是在打招呼",
		"，不需要调用任何工具，直",
		"接回复即可。\n",
		"</think>\n\n",
		"你好呀！😊",
		"很高兴见到你！有什么我可以帮你的吗？无论是查天气、搜索信息，还是聊聊天，我都随时为你服务～",
	}

	var c *regexp.Regexp
	var regex, replacement string

	Regex := `"(?i)I do not engage .+:\n":""`

	compile := regexp.MustCompile(`"(.+)" *: *"(.*)"`, regexp.ECMAScript)
	matched, err := compile.FindStringMatch(Regex)
	if err != nil {
		t.Fatal(err)
	}
	regex, replacement = matched.GroupByNumber(1).String(), matched.GroupByNumber(2).String()
	c = regexp.MustCompile(regex, regexp.ECMAScript)

	H := func(index int, content string) (state int, result string) {
		if Regex == "blank" {
			return Matched, ""
		}

		result, err := c.Replace(content, replacement, 0, 1)
		if content != "" {
			t.Logf("ctx.Put(ThinkReason, result): %s", result)
			return Matched, ""
		}

		if err != nil {
			t.Errorf("compile failed: %s\n         error: %v", regex, err)
			return Matched, content
		}
		return Matched, result
	}

	think := 7
	var once sync.Once
	interceptors := []Interceptor{
		&symbolInterceptor{
			Find: "<think>",
			H: func(_ int, content string) (int, string) {
				once.Do(func() {
					if idx := strings.Index(content, "<think>"); idx > 0 {
						think += idx
					}
				})

				defer func() { think = len(content) }()
				idx := strings.Index(content, "</think>")
				if idx < 0 {
					t.Logf("ctx.Put(ThinkReason, content): %s", content[think:])
					return Matching, content
				}

				splitter := idx + len("</think>")
				t.Logf("ctx.Put(ThinkReason, content[:idx]): %s", content[think:idx])
				return Matched, content[splitter:]
			},
		},
		&markupInterceptor{
			Find: "I do not engage",
			Over: "\n",
			H:    H,
		},
		&markupInterceptor{
			Find: "<tool_call>",
			Over: "</tool_call>",
			H: func(index int, content string) (state int, result string) {
				content = strings.TrimSpace(content)
				t.Logf("ctx.Put(ToolCall, content[11:len(content)-12]): %s", content[11:len(content)-12])
				return Matched, ""
			},
		},
		&symbolInterceptor{
			Find: "<|im_end|>",
			H: func(_ int, content string) (state int, result string) {
				state = Matched
				return
			},
		},
	}

	for _, block := range chunk {
		result := executeInterceptors(interceptors, block, false)
		t.Log(result)
	}
	result := executeInterceptors(interceptors, "", true)
	t.Log(result)
}

func executeInterceptors(matchers []Interceptor, chunk string, over bool) string {
	state := Default
	for _, mat := range matchers {
		state, chunk = mat.scan(chunk, over)
		if state != Default {
			break
		}
	}
	return chunk
}
