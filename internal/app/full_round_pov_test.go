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
				EndReason:       demo.FullRoundPOVEndRoundEnd,
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

func TestBuildFullRoundPOVSegmentsForPlugin_UsesFixedOneSecondEndPadding(t *testing.T) {
	plan := &demo.FullRoundPOVPlan{
		PlayerName:    "target",
		PlayerSteamID: "76561190000000001",
		Segments: []demo.FullRoundPOVSegment{
			{
				Round:              1,
				RecordStartTick:    100,
				RecordEndTick:      980,
				RoundEndTick:       1600,
				NextRoundStartTick: 2000,
				TargetSlot:         12,
				EndReason:          demo.FullRoundPOVEndTargetDeath,
			},
			{
				Round:              2,
				RecordStartTick:    2000,
				RecordEndTick:      3000,
				RoundEndTick:       3050,
				NextRoundStartTick: 4000,
				TargetSlot:         12,
				EndReason:          demo.FullRoundPOVEndTargetDeath,
			},
			{
				Round:              3,
				RecordStartTick:    4000,
				RecordEndTick:      5000,
				RoundEndTick:       5600,
				NextRoundStartTick: 6000,
				TargetSlot:         12,
				EndReason:          demo.FullRoundPOVEndRoundEnd,
			},
		},
	}

	segments := buildFullRoundPOVSegmentsForPlugin(plan, ClipSettings{
		KillerPostSeconds: 2,
	}, 64)

	if len(segments) != 3 {
		t.Fatalf("segments len=%d want 3", len(segments))
	}
	if segments[0].EndTick != 1044 {
		t.Fatalf("death end tick=%d want 1044", segments[0].EndTick)
	}
	if segments[1].EndTick != 3064 {
		t.Fatalf("death end tick should ignore killer_post_seconds and round end, got %d want 3064", segments[1].EndTick)
	}
	if segments[2].EndTick != 5936 {
		t.Fatalf("round-end segment should end one second before next round start, got %d want 5936", segments[2].EndTick)
	}
}
