package plugingen

import (
	"testing"

	"cs2-highlight-tool-v2/internal/clipsjson"
	"cs2-highlight-tool-v2/internal/demo"
)

func boolPtr(b bool) *bool { return &b }

func TestFilterItemsByHistory_ReturnsNilWhenNoItems(t *testing.T) {
	result := FilterItemsByHistory(nil, []TakePlan{{DemoPath: "a", View: "killer", SpecMode: 1, KillIDs: []string{"k1"}}}, nil)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestFilterItemsByHistory_ReturnsNilWhenNoPlans(t *testing.T) {
	items := []clipsjson.Item{
		{Kill: demo.ClipKill{ID: "k1"}, IncludeKiller: boolPtr(true)},
	}
	result := FilterItemsByHistory(items, nil, nil)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestFilterItemsByHistory_ExcludesItemsAlreadyInHistory(t *testing.T) {
	items := []clipsjson.Item{
		{Kill: demo.ClipKill{ID: "k1"}, IncludeKiller: boolPtr(true), IncludeVictim: false},
	}
	plans := []TakePlan{
		{DemoPath: "demo.dem", View: "killer", SpecMode: 1, KillIDs: []string{"k1"}},
	}
	historyKey := BuildProduceHistoryKey("demo.dem", "killer", 1, []string{"k1"})
	historyKeys := map[string]struct{}{historyKey: {}}

	result := FilterItemsByHistory(items, plans, historyKeys)
	if len(result) != 0 {
		t.Fatalf("expected empty result when all items are in history, got %d", len(result))
	}
}

func TestFilterItemsByHistory_IncludesItemsNotInHistory(t *testing.T) {
	items := []clipsjson.Item{
		{Kill: demo.ClipKill{ID: "k1"}, IncludeKiller: boolPtr(true), IncludeVictim: false},
		{Kill: demo.ClipKill{ID: "k2"}, IncludeKiller: boolPtr(true), IncludeVictim: false},
	}
	plans := []TakePlan{
		{DemoPath: "demo.dem", View: "killer", SpecMode: 1, KillIDs: []string{"k1"}},
		{DemoPath: "demo.dem", View: "killer", SpecMode: 1, KillIDs: []string{"k2"}},
	}
	// Only k1's plan is in history
	historyKey := BuildProduceHistoryKey("demo.dem", "killer", 1, []string{"k1"})
	historyKeys := map[string]struct{}{historyKey: {}}

	result := FilterItemsByHistory(items, plans, historyKeys)
	if len(result) != 1 {
		t.Fatalf("expected 1 item (k2 only), got %d", len(result))
	}
	if result[0].Kill.ID != "k2" {
		t.Fatalf("expected k2, got %q", result[0].Kill.ID)
	}
}

func TestFilterItemsByHistory_SeparatesKillerAndVictimViews(t *testing.T) {
	items := []clipsjson.Item{
		{Kill: demo.ClipKill{ID: "k1"}, IncludeKiller: boolPtr(true), IncludeVictim: true},
	}
	plans := []TakePlan{
		{DemoPath: "demo.dem", View: "killer", SpecMode: 1, KillIDs: []string{"k1"}},
		{DemoPath: "demo.dem", View: "victim", SpecMode: 1, KillIDs: []string{"k1"}},
	}
	// Only killer plan is in history
	killerKey := BuildProduceHistoryKey("demo.dem", "killer", 1, []string{"k1"})
	historyKeys := map[string]struct{}{killerKey: {}}

	result := FilterItemsByHistory(items, plans, historyKeys)
	if len(result) != 1 {
		t.Fatalf("expected 1 item (victim view only), got %d", len(result))
	}
	if result[0].IncludeKiller == nil || *result[0].IncludeKiller {
		t.Fatalf("expected IncludeKiller=false for victim-only item, got %v", result[0].IncludeKiller)
	}
	if !result[0].IncludeVictim {
		t.Fatal("expected IncludeVictim=true for victim-only item")
	}
}

func TestFilterItemsByHistory_IgnoresNonMatchingHistoryKeys(t *testing.T) {
	items := []clipsjson.Item{
		{Kill: demo.ClipKill{ID: "k1"}, IncludeKiller: boolPtr(true)},
	}
	plans := []TakePlan{
		{DemoPath: "demo.dem", View: "killer", SpecMode: 1, KillIDs: []string{"k1"}},
	}
	// Different history key (edited video) that should not affect produce filtering
	historyKeys := map[string]struct{}{
		"edited#123456#d:/clips/edit.mp4": {},
	}

	result := FilterItemsByHistory(items, plans, historyKeys)
	if len(result) != 1 {
		t.Fatalf("expected 1 item (not in history), got %d", len(result))
	}
	if result[0].Kill.ID != "k1" {
		t.Fatalf("unexpected item: %+v", result[0])
	}
}
