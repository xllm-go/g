//go:build ignore

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "embed"
)

var (
	//go:embed patch
	patch []byte
)

func main() {
	module := "github.com/go-rod/rod@v0.116.2"
	dir := getModule(module)
	file := filepath.Join(dir, "lib/launcher/browser.go")
	check(file)
	write(file)
}

func getModule(module string) (dir string) {
	// 执行命令: go list -m -f '{{.Dir}}' github.com/gin-gonic/gin
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	dir = strings.TrimSpace(out.String())
	if dir == "" {
		panic("not found module: " + module)
	}

	return
}

func write(file string) {
	err := os.WriteFile(file, patch, 0644)
	if err != nil {
		panic(err)
	}
}

func check(file string) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		panic("file not found: " + file)
	}
	err := os.Chmod(file, 0644)
	if err != nil {
		panic(err)
	}
}
