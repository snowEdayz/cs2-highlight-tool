package appdata

import (
	"path/filepath"
	"runtime"
	"strings"
)

const AppDataDirName = "CS2 Highlight Tool"

type Paths struct {
	ExeDir  string
	DataDir string
}

// ResolveExeOnly 返回 {ExeDir, DataDir: ""}。
// DataDir 由 App 层从注册表 (Windows) 或兜底逻辑（非 Windows）填入。
func ResolveExeOnly(exeDir string) Paths {
	return Paths{
		ExeDir:  filepath.Clean(strings.TrimSpace(exeDir)),
		DataDir: "",
	}
}

// samePath 跨平台路径比较：Windows 忽略大小写。
func samePath(a string, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
