package interceptor

import (
	"github.com/xllm-go/g/logger"
)

type expressionInterceptor struct {
	strategy *symbolInterceptor

	Find string
	Max  int
	H    func(index int, content string) (state int, result string)
}

func (interceptor *expressionInterceptor) scan(content string, over bool) (state int, result string) {
	if interceptor.strategy == nil {
		if interceptor.Max == 0 {
			interceptor.Max = 10
		}

		H := interceptor.H
		if H != nil {
			H = func(index int, content string) (state int, result string) {
				chunk := []rune(content)
				if index+interceptor.Max > len(chunk)-1 {
					return Matching, content
				}
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
