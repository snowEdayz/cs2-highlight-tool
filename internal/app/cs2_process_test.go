package app

import (
	"strings"
	"testing"
	"time"
)

func TestWaitForNewCS2PID_ReturnsNewestNewPID(t *testing.T) {
	old := listCS2PIDsFn
	calls := 0
	listCS2PIDsFn = func() ([]int, error) {
		calls++
		if calls == 1 {
			return []int{1000}, nil
		}
		return []int{1000, 1001, 1005}, nil
	}
	t.Cleanup(func() {
		listCS2PIDsFn = old
	})

	pid, err := waitForNewCS2PID(map[int]struct{}{1000: {}}, 300*time.Millisecond, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForNewCS2PID: %v", err)
	}
	if pid != 1005 {
		t.Fatalf("pid=%d want 1005", pid)
	}
}

func TestWaitForNewCS2PID_Timeout(t *testing.T) {
	old := listCS2PIDsFn
	listCS2PIDsFn = func() ([]int, error) {
		return []int{1000}, nil
	}
	t.Cleanup(func() {
		listCS2PIDsFn = old
	})

	_, err := waitForNewCS2PID(map[int]struct{}{1000: {}}, 50*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Fatalf("unexpected error: %v", err)
	}
}
