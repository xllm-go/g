package interceptor

import (
	"strings"

	"github.com/xllm-go/g/logger"
)

type markupInterceptor struct {
	strategy *symbolInterceptor

	Find string
	Over string
	H    func(index int, content string) (state int, result string)
}

func (interceptor *markupInterceptor) scan(content string, over bool) (state int, result string) {
	if interceptor.strategy == nil {
		H := interceptor.H
		if H != nil {
			H = func(index int, content string) (state int, result string) {
				idx := strings.LastIndex(content, interceptor.Over)
				if idx < 0 {
					return Matching, content
				}

				splitter := idx + len(interceptor.Over)
				interceptor.strategy.cache = content[splitter:]
				content = content[:splitter]
				logger.Sugar().Infof("execute matcher[%s] content:\n%s", interceptor.strategy.Find, content)
				return interceptor.H(index, content)
			}
		}

		interceptor.strategy = &symbolInterceptor{
			Find: interceptor.Find,
			H:    H,
		}
	}

	return interceptor.strategy.scan(content, over)
}
