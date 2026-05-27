package app

import (
	"fmt"
	"time"
)

const (
	cs2ProcessDetectTimeout      = 30 * time.Second
	cs2ProcessDetectPollInterval = 250 * time.Millisecond
	cs2ProcessCloseGraceTimeout  = 5 * time.Second
)

var (
	listCS2PIDsFn          = listCS2PIDs
	closeCS2ProcessByPIDFn = closeCS2ProcessByPID
)

func snapshotPIDSet(pids []int) map[int]struct{} {
	result := make(map[int]struct{}, len(pids))
	for _, pid := range pids {
		if pid <= 0 {
			continue
		}
		result[pid] = struct{}{}
	}
	return result
}

func waitForNewCS2PID(before map[int]struct{}, timeout time.Duration, pollInterval time.Duration) (int, error) {
	if before == nil {
		before = map[int]struct{}{}
	}
	if timeout <= 0 {
		timeout = cs2ProcessDetectTimeout
	}
	if pollInterval <= 0 {
		pollInterval = cs2ProcessDetectPollInterval
	}

	deadline := time.Now().Add(timeout)
	for {
		pids, err := listCS2PIDsFn()
		if err != nil {
			return 0, err
		}

		newestPID := 0
		for _, pid := range pids {
			if pid <= 0 {
				continue
			}
			if _, exists := before[pid]; exists {
				continue
			}
			if pid > newestPID {
				newestPID = pid
			}
		}
		if newestPID > 0 {
			return newestPID, nil
		}

		if time.Now().After(deadline) {
			return 0, fmt.Errorf("等待新的 cs2.exe 进程超时")
		}
		time.Sleep(pollInterval)
	}
}
