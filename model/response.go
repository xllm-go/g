package model

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type chunkExpr int8

const EOF = "[EOF]"

type FuncCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"arguments"`
}

type ChunkBodies struct {
	Chunk    string
	Think    string
	Function *FuncCall
}

func (bodies ChunkBodies) expr() chunkExpr {
	if bodies.Function != nil {
		return 0
	}
	return 1
}

func CreateChunk(chunk, think string) *ChunkBodies {
	return &ChunkBodies{
		Chunk: chunk,
		Think: think,
	}
}

func CreateFunction(name string, arguments json.RawMessage) *ChunkBodies {
	return &ChunkBodies{
		Function: &FuncCall{
			Name: name,
			Args: arguments,
		},
	}
}

func CreateStreamResponse(chunkBodies *ChunkBodies, created int64) (response *Response) {
	response = &Response{
		Model:   "LLM",
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion.chunk",
	}

	switch chunkBodies.expr() {
	case 0:
		response.Choices = []Choice{
			{
				Index: 0,
				Delta: &struct {
					Type             string `json:"type,omitempty"`
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{"text", "assistant", "", "", []ChoiceToolCall{
					ChoiceToolCall{
						"index": 0,
						"Function": Record[string, string]{
							"name":      chunkBodies.Function.Name,
							"arguments": string(chunkBodies.Function.Args),
						},
					}.
						Lambda().E(chunkBodies.Function.Name).Put("id", "call_"+hex(5)).
						Lambda().E(chunkBodies.Function.Name).Put("type", "Function"),
				}},
			},
		}

	case 1:
		response.Choices = []Choice{
			{
				Index: 0,
				Delta: &struct {
					Type             string `json:"type,omitempty"`
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{"text", "assistant", chunkBodies.Chunk, chunkBodies.Think, nil},
			},
		}
	}

	return
}

func CreateResponse(chunkBodies *ChunkBodies, created int64) (response *Response) {
	stop := "stop"

	var toolCalls []ChoiceToolCall
	if chunkBodies.Function != nil {
		toolCalls = append(toolCalls, ChoiceToolCall{
			"index": 0,
			"type":  "Function",
			"id":    "call_" + hex(5),
			"function": Record[string, string]{
				"name":      chunkBodies.Function.Name,
				"arguments": string(chunkBodies.Function.Args),
			},
		})
	}

	response = &Response{
		Model:   "LLM",
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion",
		Choices: []Choice{
			{
				Index: 0,
				Message: &struct {
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{"assistant", chunkBodies.Chunk, chunkBodies.Think, toolCalls},
				FinishReason: &stop,
			},
		},
		//Usage: usage,
	}

	return
}

func SplitEach(content string, w func(chunk string)) {
	pos := 0
	runeStr := []rune(content)
	step := 30

	for {
		contentL := len(runeStr[pos:])
		if contentL > step {
			w(string(runeStr[pos : pos+step]))
			pos += step
			continue
		}

		w(string(runeStr[pos:]))
		time.Sleep(100 * time.Millisecond)
		break
	}
}

func hex(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	bytes := make([]rune, n)
	for i := range bytes {
		bytes[i] = runes[r.Intn(len(runes))]
	}
	return string(bytes)
}
