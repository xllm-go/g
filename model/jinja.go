package model

import (
	"strings"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

func JinjaMessage(jinja string, completion *Completion) (v string, err error) {
	engine, err := gonja.FromString(jinja)
	if err != nil {
		return
	}

	w := new(strings.Builder)
	err = engine.Execute(w, exec.NewContext(
		map[string]interface{}{
			"messages": completion.Messages,
			"tools":    completion.Tools,
		}))
	if err != nil {
		return
	}

	v = w.String()
	return
}
