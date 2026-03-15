package interceptor

// 字符块匹配器，只向后匹配
type symbolInterceptor struct {
	cache string // 缓存的字符
	Find  string // 字符块匹配前置，'*'则匹配任意
	// 具体的匹配实现, cache 仅在 Matched 状态有效
	H func(index int, content string) (state int, result string)
}

func (interceptor *symbolInterceptor) scan(content string, over bool) (state int, result string) {
	state = Default
	index := 0

	if interceptor.Find == "" {
		state = Matched
		goto label
	}

	content = interceptor.cache + content
	index, state = interceptor.nextToken(content)

label:
	// 没有命中，返回所有内容（包括cache）
	if state == Default {
		interceptor.cache = ""
		result = content
		return
	}

	// 还在匹配中，再次校验是否命中
	if state == Matching {
		interceptor.cache = content // 缓存, 等待下次输入
		result = ""
		return
	}

	// 执行上游拦截处理
	return interceptor.execute(content, over, index)
}

func (interceptor *symbolInterceptor) nextToken(content string) (index, state int) {
	var (
		chunk = []rune(content)
		find  = []rune(interceptor.Find)

		pos = 0
	)

	state = Default
	for index = range chunk {
		var char rune
		// 到这里就代表命中了，推出检索
		if len(find) == pos {
			state = Matched
			if interceptor.H != nil {
				break
			}
			return
		}

		// 索引的字符不一致，重置累加
		char = find[pos]
		//logger.Sugar().Infof("[interceptor] char:[%v]%s, %v-%d, %s",
		//	interceptor.Find,
		//	string(find[pos]),
		//	char == chunk[index],
		//	index, string(chunk[index]),
		//)
		if char != chunk[index] {
			pos = 0
			state = Default
			continue
		}

		// 索引的字符一致，累加下一个索引
		pos++
		state = Matching
		continue
	}

	return
}

func (interceptor *symbolInterceptor) execute(content string, over bool, index int) (int, string) {
	// 命中，但是上游没有提供处理逻辑
	if interceptor.H == nil {
		interceptor.cache = ""
		return Matched, content
	}

	// 执行上游自定义处理
	state, result := interceptor.H(index, content)
	// 被标记为处理完毕
	if state == Matched {
		return state, result
	}

	// 上游应当是拒绝处理了
	if state != Matching {
		return state, content
	}

	// 已经没有后续输入了
	if over {
		return Default, content
	}

	// 被标记为处理中
	interceptor.cache = result
	return state, ""

}
