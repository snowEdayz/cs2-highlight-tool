//go:build !windows

package app

// EnsureSingleInstance 在非 Windows 平台上为空操作，始终返回成功。
func EnsureSingleInstance() (cleanup func(), err error) {
	return func() {}, nil
}
