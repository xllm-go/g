package internal

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xllm-go/g/env"
	"github.com/xllm-go/g/internal/v1"
	"github.com/xllm-go/g/logger"
)

var (
	cArgs = &CobraArgs{
		Port:     7860,
		LogLevel: "info",
		LogPath:  "log",
	}

	cmd = &cobra.Command{
		Use:     "adapter",
		Version: "v3.0.1-beta",
		Short:   "GPT接口适配器",
		Long:    "GPT接口适配器。统一适配接口规范，集成了bing、claude，gemini...\n项目地址: https://github.com/bincooo/chatgpt-adapter",

		Run: func(cmd *cobra.Command, args []string) {
			// init
			logger.InitLogger(
				cArgs.LogPath,
				LogLevel(cArgs.LogLevel),
			)

			err := env.InitEnviron()
			if err != nil {
				logger.Sugar().Fatalf("config.yaml is not exists; %v", err)
			}

			if port := env.Env.GetInt("server.port"); port > 0 {
				cArgs.Port = port
			}

			env.Initialized()
			if cArgs.MView {
				println("模型可用列表:")
				var hasModel = false
				for model := range v1.Models() {
					println("    - " + model.Id)
					hasModel = true
				}
				if !hasModel {
					println("    - 空 -")
				}
				return
			}

			v1.Initialized(fmt.Sprintf(":%d", cArgs.Port))
		},
	}
)

type CobraArgs struct {
	Port     int    // 服务端口 port
	LogLevel string // 日志级别: debug|info|warn|error
	LogPath  string // ogger path
	Proxied  string // 本地代理 proxies
	MView    bool   // 展示模型列表
	Hook     bool   // 鼠标助手
}

func Execute() {
	cmd.Flags().IntVarP(&cArgs.Port, "port", "p", 8080, "服务端口 port")
	cmd.Flags().StringVarP(&cArgs.LogLevel, "logger", "L", "debug", "日志级别: debug|info|warn|error")
	cmd.Flags().StringVar(&cArgs.LogPath, "logger-path", "", "日志路径 logger path")
	cmd.Flags().StringVarP(&cArgs.Proxied, "proxies", "P", "", "本地代理 proxies")
	cmd.Flags().BoolVarP(&cArgs.MView, "models", "M", false, "展示模型列表")
	cmd.Flags().BoolVarP(&cArgs.Hook, "hook", "H", false, "鼠标助手")
	if err := cmd.Execute(); err != nil {
		println(err.Error())
	}
}

func LogLevel(lv string) logger.Level {
	switch lv {
	case "debug":
		return logger.DebugLevel
	case "warn":
		return logger.WarnLevel
	case "error":
		return logger.ErrorLevel
	default:
		return logger.InfoLevel
	}
}

func getArgs() *CobraArgs {
	return cArgs
}
