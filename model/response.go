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

	Err error

	Stream bool
}

func (bodies ChunkBodies) expr() chunkExpr {
	if bodies.Err != nil {
		return -1
	}
	if bodies.Function != nil {
		return 0
	}
	return 1
}

func CreateFunction(chunk string, stream bool) *ChunkBodies {
	var fun FuncCall
	_ = json.Unmarshal([]byte(chunk), &fun)
	return &ChunkBodies{
		Stream:   stream,
		Function: &fun,
	}
}

func CreateResponse(chunkBodies *ChunkBodies, unix int64) *Response {
	if chunkBodies.Stream {
		return createStreamResponse(chunkBodies, unix)
	}

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

	if chunkBodies.Err != nil {
		return &Response{
			Model:   "LLM",
			Created: unix,
			Id:      fmt.Sprintf("chatcmpl-%d", unix),
			Object:  "chat.completion",
			Choices: []Choice{
				{
					Index: 0,
					Message: &struct {
						Role             string `json:"role,omitempty"`
						Content          string `json:"content,omitempty"`
						ReasoningContent string `json:"reasoning_content,omitempty"`

						ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
					}{"assistant", fmt.Sprintf("error: %v", chunkBodies.Err), "", nil},
					FinishReason: &stop,
				},
			},
			//Usage: usage,
		}
	}

	return &Response{
		Model:   "LLM",
		Created: unix,
		Id:      fmt.Sprintf("chatcmpl-%d", unix),
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
}

func createStreamResponse(chunkBodies *ChunkBodies, unix int64) (response *Response) {
	response = &Response{
		Model:   "LLM",
		Created: unix,
		Id:      fmt.Sprintf("chatcmpl-%d", unix),
		Object:  "chat.completion.chunk",
	}

	switch chunkBodies.expr() {
	case -1:
		response.Choices = []Choice{
			{
				Index: 0,
				Delta: &struct {
					Type             string `json:"type,omitempty"`
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{"text", "assistant", fmt.Sprintf("error: %v", chunkBodies.Err), "", nil},
			},
		}
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
				}{Role: "assistant", ToolCalls: []ChoiceToolCall{
					ChoiceToolCall{
						"index": 0,
						"function": Record[string, string]{
							"name":      chunkBodies.Function.Name,
							"arguments": string(chunkBodies.Function.Args),
						},
					}.
						Lambda().E(chunkBodies.Function.Name).Put("id", "call_"+hex(5)).
						Lambda().E(chunkBodies.Function.Name).Put("type", "function"),
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

func createStopResponse(stop string, unix int64) (response *Response) {
	return &Response{
		Model:   "LLM",
		Created: unix,
		Id:      fmt.Sprintf("chatcmpl-%d", unix),
		Object:  "chat.completion",
		Choices: []Choice{
			{
				Index: 0,
				Delta: &struct {
					Type             string `json:"type,omitempty"`
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{},
				FinishReason: &stop,
			},
		},
		//Usage: usage,
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
