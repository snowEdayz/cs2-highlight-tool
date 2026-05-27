//go:build !windows

package app

func listPIDsByExeName(_ string) ([]int, error) {
	return nil, nil
}

func sendWMCloseToPlatformClient(_ int) error {
	return nil
}
