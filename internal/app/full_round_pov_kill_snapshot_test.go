package app

import (
	"testing"

	"cs2-highlight-tool-v2/internal/clipsjson"
	"cs2-highlight-tool-v2/internal/demo"
)

// TestRegisterProduceKillSnapshot_RegistersVictimClipKills verifies that kill
// info carried by the user-selected clip items (killer / victim perspectives)
// is folded into the kill snapshot store keyed by demo path. This is the
// baseline behaviour that the full_round_pov enrichment builds on top of.
func TestRegisterProduceKillSnapshot_RegistersVictimClipKills(t *testing.T) {
	store := make(map[string]map[string]demo.ClipKill)
	items := []clipsjson.Item{
		{Kill: demo.ClipKill{ID: "k1", Round: 1, KillerSteamID: "p1"}, IncludeVictim: true},
	}
	plans := []ProduceTakePlan{
		{DemoPath: "demoA.dem", TakeIndex: 1, View: "victim", KillIDs: []string{"k1"}},
	}

	registerProduceKillSnapshot(store, plans, items, "")

	got := store["demoA.dem"]
	if got == nil {
		t.Fatalf("expected kill snapshot for demoA.dem")
	}
	if _, ok := got["k1"]; !ok {
		t.Fatalf("expected kill k1 registered, got %+v", got)
	}
}

// TestRegisterProduceKillSnapshot_SkipsEmptyItemsAndPlans guards against the
// store being mutated (or panicking) when both items and POV plans are empty.
func TestRegisterProduceKillSnapshot_SkipsEmptyItemsAndPlans(t *testing.T) {
	store := make(map[string]map[string]demo.ClipKill)
	registerProduceKillSnapshot(store, nil, nil, "demoA.dem")
	if len(store) != 0 {
		t.Fatalf("store should remain empty, got %+v", store)
	}
}

// TestCollectFullRoundPOVKills_GracefulOnMissingDemo verifies that a missing or
// unreadable demo file does not break the kill snapshot build. The function
// must simply yield no POV kills for that demo path.
func TestCollectFullRoundPOVKills_GracefulOnMissingDemo(t *testing.T) {
	plans := []ProduceTakePlan{
		{DemoPath: "definitely-missing.dem", View: "full_round_pov", Round: 1, PlayerSteamID: "765"},
		{DemoPath: "definitely-missing.dem", View: "full_round_pov", Round: 2, PlayerSteamID: "765"},
	}
	result := collectFullRoundPOVKills(plans)
	if len(result) != 0 {
		t.Fatalf("missing demo should yield no POV kills, got %+v", result)
	}
}

// TestCollectFullRoundPOVKills_IgnoresNonPOVPlans verifies that killer / victim
// take plans are ignored by the POV-only kill resolver.
func TestCollectFullRoundPOVKills_IgnoresNonPOVPlans(t *testing.T) {
	plans := []ProduceTakePlan{
		{DemoPath: "demoA.dem", View: "killer", Round: 1, PlayerSteamID: "765"},
		{DemoPath: "demoA.dem", View: "victim", Round: 1, PlayerSteamID: "765"},
	}
	result := collectFullRoundPOVKills(plans)
	if len(result) != 0 {
		t.Fatalf("non-POV plans should yield no POV kills, got %+v", result)
	}
}

// TestCollectFullRoundPOVKills_SkipsIncompletePlans ensures plans that lack a
// tracked player SteamID, demo path, or valid round number are skipped so they
// cannot accidentally match every round / every player.
func TestCollectFullRoundPOVKills_SkipsIncompletePlans(t *testing.T) {
	plans := []ProduceTakePlan{
		{DemoPath: "", View: "full_round_pov", Round: 1, PlayerSteamID: "765"},
		{DemoPath: "demoA.dem", View: "full_round_pov", Round: 1, PlayerSteamID: ""},
		{DemoPath: "demoA.dem", View: "full_round_pov", Round: 0, PlayerSteamID: "765"},
		{DemoPath: "demoA.dem", View: "full_round_pov", Round: -3, PlayerSteamID: "765"},
	}
	result := collectFullRoundPOVKills(plans)
	if len(result) != 0 {
		t.Fatalf("incomplete POV plans should yield no POV kills, got %+v", result)
	}
}
