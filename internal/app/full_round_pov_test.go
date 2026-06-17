package app

import (
	"testing"

	"cs2-highlight-tool-v2/internal/demo"
)

func TestBuildFullRoundPOVSegmentsForPlugin_UsesRecordTicksAndTargetSlot(t *testing.T) {
	plan := &demo.FullRoundPOVPlan{
		PlayerName:    "target",
		PlayerSteamID: "76561190000000001",
		Segments: []demo.FullRoundPOVSegment{
			{
				Round:           1,
				RecordStartTick: 420,
				RecordEndTick:   980,
				TargetSlot:      12,
				EndReason:       demo.FullRoundPOVEndTargetDeath,
			},
		},
	}

	segments := buildFullRoundPOVSegmentsForPlugin(plan, ClipSettings{
		EnableVoice:        true,
		EnableSpecShowXray: true,
	}, 64)

	if len(segments) != 1 {
		t.Fatalf("segments len=%d want 1", len(segments))
	}
	seg := segments[0]
	if seg.StartTick != 420 || seg.EndTick != 980 || seg.Target != "12" {
		t.Fatalf("unexpected ticks/target: %+v", seg)
	}
	if seg.PlayerName != "target" || seg.PlayerSteamID != "76561190000000001" || seg.Round != 1 {
		t.Fatalf("missing player metadata: %+v", seg)
	}
}

func TestBuildFullRoundPOVSegmentsForPlugin_ExtendsDeathEndByKillerPostSeconds(t *testing.T) {
	plan := &demo.FullRoundPOVPlan{
		PlayerName:    "target",
		PlayerSteamID: "76561190000000001",
		Segments: []demo.FullRoundPOVSegment{
			{
				Round:           1,
				RecordStartTick: 100,
				RecordEndTick:   980,
				RoundEndTick:    1600,
				TargetSlot:      12,
				EndReason:       demo.FullRoundPOVEndTargetDeath,
			},
			{
				Round:           2,
				RecordStartTick: 2000,
				RecordEndTick:   3000,
				RoundEndTick:    3050,
				TargetSlot:      12,
				EndReason:       demo.FullRoundPOVEndTargetDeath,
			},
			{
				Round:           3,
				RecordStartTick: 4000,
				RecordEndTick:   5000,
				RoundEndTick:    5600,
				TargetSlot:      12,
				EndReason:       demo.FullRoundPOVEndRoundEnd,
			},
		},
	}

	segments := buildFullRoundPOVSegmentsForPlugin(plan, ClipSettings{
		KillerPostSeconds: 2,
	}, 64)

	if len(segments) != 3 {
		t.Fatalf("segments len=%d want 3", len(segments))
	}
	if segments[0].EndTick != 1108 {
		t.Fatalf("death end tick=%d want 1108", segments[0].EndTick)
	}
	if segments[1].EndTick != 3050 {
		t.Fatalf("clamped death end tick=%d want 3050", segments[1].EndTick)
	}
	if segments[2].EndTick != 5000 {
		t.Fatalf("round-end segment should keep end tick, got %d", segments[2].EndTick)
	}
}
