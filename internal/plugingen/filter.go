package plugingen

import (
	"strings"

	"cs2-highlight-tool-v2/internal/clipsjson"
)

// TakePlan describes a single recording take — the demo file, camera view,
// spec mode, and the set of kill IDs included in that take.
// This mirrors the app-layer ProduceTakePlan but is defined here so that
// pure filtering logic has no dependency on the Wails binding layer.
type TakePlan struct {
	DemoPath  string
	View      string
	SpecMode  int
	KillIDs   []string
}

// FilterItemsByHistory returns the subset of items whose corresponding take
// plans are NOT already present in historyKeys. Items are also adjusted so
// that only the needed killer/victim perspectives are included.
//
// historyKeys is a set of keys produced by BuildProduceHistoryKey.
func FilterItemsByHistory(
	items []clipsjson.Item,
	plans []TakePlan,
	historyKeys map[string]struct{},
) []clipsjson.Item {
	if len(items) == 0 || len(plans) == 0 {
		return nil
	}
	killerKeepByKillID := make(map[string]bool)
	victimKeepByKillID := make(map[string]bool)
	for _, plan := range plans {
		if _, exists := historyKeys[BuildProduceHistoryKey(plan.DemoPath, plan.View, plan.SpecMode, plan.KillIDs)]; exists {
			continue
		}
		view := strings.ToLower(strings.TrimSpace(plan.View))
		for _, killID := range plan.KillIDs {
			id := strings.TrimSpace(killID)
			if id == "" {
				continue
			}
			if view == "victim" {
				victimKeepByKillID[id] = true
			} else {
				killerKeepByKillID[id] = true
			}
		}
	}

	filtered := make([]clipsjson.Item, 0, len(items))
	for _, item := range items {
		killID := strings.TrimSpace(item.Kill.ID)
		if killID == "" {
			continue
		}
		keepKiller := killerKeepByKillID[killID]
		keepVictim := victimKeepByKillID[killID]
		if !keepKiller && !keepVictim {
			continue
		}
		next := item
		v := keepKiller
		next.IncludeKiller = &v
		next.IncludeVictim = keepVictim
		filtered = append(filtered, next)
	}
	return filtered
}
