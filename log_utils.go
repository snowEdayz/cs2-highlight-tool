package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	colorRed     = color.New(color.FgRed)
	colorGreen   = color.New(color.FgGreen)
	colorYellow  = color.New(color.FgYellow)
	colorBlue    = color.New(color.FgBlue)
	colorMagenta = color.New(color.FgMagenta)
	colorCyan    = color.New(color.FgCyan)
	colorWhite   = color.New(color.FgWhite)
	colorBold    = color.New(color.Bold)

	// 组合颜色
	colorMagentaBold = color.New(color.FgMagenta, color.Bold)
	colorCyanBold    = color.New(color.FgCyan, color.Bold)
	colorGreenBold   = color.New(color.FgGreen, color.Bold)
)

var appCtx context.Context

type LogMessage struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

type ProgressMessage struct {
	Active        bool    `json:"active"`
	Percent       float64 `json:"percent"`
	Indeterminate bool    `json:"indeterminate"`
}

func emitLog(level, message string) {
	if appCtx == nil {
		return
	}
	runtime.EventsEmit(appCtx, "log", LogMessage{
		Level:   level,
		Message: message,
		Time:    time.Now().Format(time.RFC3339),
	})
}

func emitProgress(active bool, percent float64, indeterminate bool) {
	if appCtx == nil {
		return
	}
	runtime.EventsEmit(appCtx, "download_progress", ProgressMessage{
		Active:        active,
		Percent:       percent,
		Indeterminate: indeterminate,
	})
}

func printSuccess(text string) {
	emitLog("success", text)
	colorGreen.Println(text)
}

func printError(text string) {
	emitLog("error", text)
	colorRed.Println(text)
}

func printWarning(text string) {
	emitLog("warning", text)
	colorYellow.Println(text)
}

func printInfo(text string) {
	emitLog("info", text)
	colorCyan.Println(text)
}

func printTitle(text string) {
	emitLog("title", text)
	colorCyanBold.Println(text)
}

func execCommandHidden(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

// 等待用户输入（用于拖拽模式）
func waitForExit() {
	fmt.Println()
	colorYellow.Print("按 Enter 键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
