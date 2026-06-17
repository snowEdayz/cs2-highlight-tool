package demo

import "testing"

func TestBuildFullRoundPOVPlan_UsesRoundStartAndDeathOrRoundEnd(t *testing.T) {
	player := PlayerInfo{Name: "target", SteamID: 76561190000000001}
	otherVictim := uint64(76561190000000002)
	rounds := []FullRoundPOVRound{
		{
			Round:         1,
			StartTick:     100,
			FreezeEndTick: 420,
			EndTick:       1600,
			Slots:         map[uint64]int{player.SteamID: 12},
			Deaths: []FullRoundPOVDeath{
				{Tick: 700, VictimSteamID: otherVictim, KillerSteamID: player.SteamID},
				{Tick: 980, VictimSteamID: player.SteamID, KillerSteamID: otherVictim},
			},
		},
		{
			Round:         2,
			StartTick:     2000,
			FreezeEndTick: 2400,
			EndTick:       3600,
			Slots:         map[uint64]int{player.SteamID: 12},
			Deaths: []FullRoundPOVDeath{
				{Tick: 3000, VictimSteamID: otherVictim, KillerSteamID: player.SteamID},
			},
		},
	}

	plan := BuildFullRoundPOVPlan(player, rounds)

	if plan.PlayerSteamID != "76561190000000001" {
		t.Fatalf("player steam id=%q", plan.PlayerSteamID)
	}
	if len(plan.Segments) != 2 {
		t.Fatalf("segments len=%d want 2", len(plan.Segments))
	}
	first := plan.Segments[0]
	if first.RecordStartTick != 100 || first.RecordEndTick != 980 || first.EndReason != "target_death" {
		t.Fatalf("unexpected first segment: %+v", first)
	}
	second := plan.Segments[1]
	if second.RecordStartTick != 2000 || second.RecordEndTick != 3600 || second.EndReason != "round_end" {
		t.Fatalf("unexpected second segment: %+v", second)
	}
}

func TestBuildFullRoundPOVPlan_IgnoresDeathsOutsideLiveRound(t *testing.T) {
	player := PlayerInfo{Name: "target", SteamID: 76561190000000001}
	otherVictim := uint64(76561190000000002)
	rounds := []FullRoundPOVRound{
		{
			Round:         1,
			StartTick:     100,
			FreezeEndTick: 420,
			EndTick:       1600,
			Slots:         map[uint64]int{player.SteamID: 7},
			Deaths: []FullRoundPOVDeath{
				{Tick: 300, VictimSteamID: player.SteamID, KillerSteamID: otherVictim},
				{Tick: 1700, VictimSteamID: player.SteamID, KillerSteamID: otherVictim},
				{Tick: 900, VictimSteamID: otherVictim, KillerSteamID: player.SteamID},
			},
		},
	}

	plan := BuildFullRoundPOVPlan(player, rounds)

	if len(plan.Segments) != 1 {
		t.Fatalf("segments len=%d want 1", len(plan.Segments))
	}
	seg := plan.Segments[0]
	if seg.RecordStartTick != 100 || seg.RecordEndTick != 1600 || seg.EndReason != "round_end" {
		t.Fatalf("unexpected segment: %+v", seg)
	}
}

func TestBuildFullRoundPOVPlan_FiltersZeroKillRounds(t *testing.T) {
	player := PlayerInfo{Name: "target", SteamID: 76561190000000001}
	otherKiller := uint64(76561190000000099)
	otherVictim := uint64(76561190000000002)
	rounds := []FullRoundPOVRound{
		{
			Round:         1,
			StartTick:     100,
			FreezeEndTick: 420,
			EndTick:       1600,
			Slots:         map[uint64]int{player.SteamID: 12},
			Deaths: []FullRoundPOVDeath{
				{Tick: 700, VictimSteamID: otherVictim, KillerSteamID: player.SteamID},
			},
		},
		{
			Round:         2,
			StartTick:     2000,
			FreezeEndTick: 2400,
			EndTick:       3600,
			Slots:         map[uint64]int{player.SteamID: 12},
			Deaths: []FullRoundPOVDeath{
				{Tick: 3000, VictimSteamID: otherVictim, KillerSteamID: otherKiller},
			},
		},
		{
			Round:         3,
			StartTick:     4000,
			FreezeEndTick: 4400,
			EndTick:       5600,
			Slots:         map[uint64]int{player.SteamID: 12},
			Deaths: []FullRoundPOVDeath{
				{Tick: 5000, VictimSteamID: otherVictim, KillerSteamID: player.SteamID},
			},
		},
	}

	plan := BuildFullRoundPOVPlan(player, rounds)

	if len(plan.Segments) != 2 {
		t.Fatalf("segments len=%d want 2", len(plan.Segments))
	}
	if plan.Segments[0].Round != 1 || plan.Segments[1].Round != 3 {
		t.Fatalf("unexpected segment rounds: %+v", plan.Segments)
	}
}
