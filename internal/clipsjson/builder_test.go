package clipsjson

import (
	"fmt"
	"strings"
	"testing"

	"cs2-highlight-tool-v2/internal/demo"
)

func TestBuild_AddsHLAEBootstrapCommands(t *testing.T) {
	result, err := Build([]Item{{
		Kill:          demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7, VictimSlot: 11},
		IncludeVictim: true,
	}}, BuildOptions{
		TickRate:           64,
		KillerPreSeconds:   1,
		KillerPostSeconds:  1,
		VictimPreSeconds:   1,
		VictimPostSeconds:  1,
		RecordFPS:          60,
		VideoPreset:        "n1",
		RecordOutputDir:    `D:/clips/output`,
		RecordBatchName:    "20260422_131500",
		EnableSpecShowXray: true,
		ActionSettings: ActionSettings{
			EnableVoiceIndices:  true,
			VoiceIndicesValue:   -1,
			EnableVoiceIndicesH: true,
			VoiceIndicesHValue:  -1,
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.Sequences) != 3 {
		t.Fatalf("sequence len=%d want 3", len(result.Sequences))
	}

	bootstrap := result.Sequences[0].Actions
	assertHasAction(t, bootstrap, "r_show_build_info 0")
	assertHasAction(t, bootstrap, "cl_trueview_show_status 0")
	assertHasAction(t, bootstrap, "engine_no_focus_sleep 0")
	assertHasAction(t, bootstrap, "cl_demo_predict 0")
	assertHasAction(t, bootstrap, "spec_show_xray 0")
	assertHasAction(t, bootstrap, "tv_listen_voice_indices -1")
	assertHasAction(t, bootstrap, "tv_listen_voice_indices_h -1")
	assertHasAction(t, bootstrap, "mirv_streams record screen enabled 1")
	assertHasAction(t, bootstrap, "mirv_streams record fps 60")
	assertHasAction(t, bootstrap, "mirv_streams record startMovieWav 1")
	assertHasPrefixAction(t, bootstrap, "mirv_streams settings add ffmpeg n1 ")
	assertHasAction(t, bootstrap, "mirv_streams record screen settings n1")
	assertHasAction(t, bootstrap, `mirv_streams record name "D:/clips/output/20260422_131500"`)
	assertHasAction(t, bootstrap, "go_to_next_sequence")
}

func TestBuild_SpecShowXrayCanBeDisabled(t *testing.T) {
	result, err := Build([]Item{{
		Kill:          demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
		IncludeVictim: false,
	}}, BuildOptions{
		TickRate:           64,
		KillerPreSeconds:   1,
		KillerPostSeconds:  1,
		RecordFPS:          60,
		VideoPreset:        "n1",
		RecordOutputDir:    `D:/clips/output`,
		RecordBatchName:    "20260422_131500",
		EnableSpecShowXray: false,
		ActionSettings:     ActionSettings{VoiceIndicesValue: 0, VoiceIndicesHValue: 0},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	assertHasAction(t, result.Sequences[0].Actions, "spec_show_xray -1")
	assertNoAction(t, result.Sequences[0].Actions, "spec_show_xray 0")
}

func TestBuild_VoiceIndicesValueFollowsEnableSwitch(t *testing.T) {
	enabled, err := Build([]Item{{
		Kill:          demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
		IncludeVictim: false,
	}}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "n1",
		RecordOutputDir:   `D:/clips/output`,
		ActionSettings: ActionSettings{
			EnableVoiceIndices:  true,
			VoiceIndicesValue:   0,
			EnableVoiceIndicesH: true,
			VoiceIndicesHValue:  0,
		},
	})
	if err != nil {
		t.Fatalf("Build enabled voice: %v", err)
	}
	assertHasAction(t, enabled.Sequences[0].Actions, "tv_listen_voice_indices -1")
	assertHasAction(t, enabled.Sequences[0].Actions, "tv_listen_voice_indices_h -1")

	disabled, err := Build([]Item{{
		Kill:          demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
		IncludeVictim: false,
	}}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "n1",
		RecordOutputDir:   `D:/clips/output`,
		ActionSettings: ActionSettings{
			EnableVoiceIndices:  false,
			VoiceIndicesValue:   -1,
			EnableVoiceIndicesH: true,
			VoiceIndicesHValue:  -1,
		},
	})
	if err != nil {
		t.Fatalf("Build disabled voice: %v", err)
	}
	assertHasAction(t, disabled.Sequences[0].Actions, "tv_listen_voice_indices 0")
	assertHasAction(t, disabled.Sequences[0].Actions, "tv_listen_voice_indices_h 0")
}

func TestBuild_AddsRecordStartEndAndTakeMetadata(t *testing.T) {
	result, err := Build([]Item{
		{Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7, VictimSlot: 11}, IncludeVictim: true},
		{Kill: demo.ClipKill{ID: "k2", Tick: 280, KillerSlot: 8, VictimSlot: 12}, IncludeVictim: true},
	}, BuildOptions{
		TickRate:           64,
		KillerPreSeconds:   1,
		KillerPostSeconds:  1,
		VictimPreSeconds:   1,
		VictimPostSeconds:  1,
		RecordFPS:          60,
		VideoPreset:        "c1",
		RecordOutputDir:    `D:/clips/output`,
		RecordBatchName:    "20260422_131500",
		EnableSpecShowXray: true,
		ActionSettings:     ActionSettings{VoiceIndicesValue: 0, VoiceIndicesHValue: 0},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var starts []Action
	var ends []Action
	for _, seq := range result.Sequences[1:] {
		for _, action := range seq.Actions {
			switch action.Cmd {
			case "mirv_streams record start":
				starts = append(starts, action)
			case "mirv_streams record end":
				ends = append(ends, action)
			}
		}
	}
	if len(starts) == 0 || len(starts) != len(ends) {
		t.Fatalf("start/end mismatch: %d/%d", len(starts), len(ends))
	}

	for idx := range starts {
		expectedTake := idx + 1
		expectedName := takeName(expectedTake)
		if starts[idx].Metadata == nil || ends[idx].Metadata == nil {
			t.Fatalf("missing metadata for take %d", expectedTake)
		}
		if starts[idx].Metadata.TakeIndex != expectedTake || ends[idx].Metadata.TakeIndex != expectedTake {
			t.Fatalf("take index mismatch for %d: start=%+v end=%+v", expectedTake, starts[idx].Metadata, ends[idx].Metadata)
		}
		if starts[idx].Metadata.TakeName != expectedName || ends[idx].Metadata.TakeName != expectedName {
			t.Fatalf("take name mismatch for %d: start=%+v end=%+v", expectedTake, starts[idx].Metadata, ends[idx].Metadata)
		}
		if starts[idx].Metadata.RecordPhase != "start" || ends[idx].Metadata.RecordPhase != "end" {
			t.Fatalf("phase mismatch for %d: start=%+v end=%+v", expectedTake, starts[idx].Metadata, ends[idx].Metadata)
		}
		if starts[idx].Tick > ends[idx].Tick {
			t.Fatalf("record start tick should <= end tick: %d > %d", starts[idx].Tick, ends[idx].Tick)
		}
	}
}

func TestBuild_TakePlansContainMergedKillIDs(t *testing.T) {
	result, err := Build([]Item{
		{Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7}, IncludeVictim: false},
		{Kill: demo.ClipKill{ID: "k2", Tick: 220, KillerSlot: 7}, IncludeVictim: false},
	}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "n1",
		RecordOutputDir:   `D:/clips/output`,
		ActionSettings:    ActionSettings{VoiceIndicesValue: 0, VoiceIndicesHValue: 0},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.TakePlans) != 1 {
		t.Fatalf("take plan len=%d want 1", len(result.TakePlans))
	}
	plan := result.TakePlans[0]
	if plan.View != "killer" {
		t.Fatalf("take view=%q want killer", plan.View)
	}
	if plan.SpecMode != 1 {
		t.Fatalf("take spec mode=%d want 1", plan.SpecMode)
	}
	if plan.TakeIndex != 1 || plan.TakeName != "take0000" {
		t.Fatalf("unexpected take identity: %+v", plan)
	}
	if len(plan.KillIDs) != 2 || plan.KillIDs[0] != "k1" || plan.KillIDs[1] != "k2" {
		t.Fatalf("unexpected kill ids: %+v", plan.KillIDs)
	}
}

func TestBuild_DifferentPerClipParamsDoNotMergeKillerSegments(t *testing.T) {
	result, err := Build([]Item{
		{
			Kill:               demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
			IncludeVictim:      false,
			EnableVoice:        true,
			EnableSpecShowXray: true,
		},
		{
			Kill:               demo.ClipKill{ID: "k2", Tick: 220, KillerSlot: 7},
			IncludeVictim:      false,
			EnableVoice:        false,
			EnableSpecShowXray: true,
			HasVoiceOverride:   true,
		},
	}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "n1",
		RecordOutputDir:   `D:/clips/output`,
		ActionSettings:    ActionSettings{VoiceIndicesValue: 0, VoiceIndicesHValue: 0},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.TakePlans) != 2 {
		t.Fatalf("take plan len=%d want 2", len(result.TakePlans))
	}
}

func TestBuild_InjectsPerPassVoiceAndXrayCommandsWhenForced(t *testing.T) {
	result, err := Build([]Item{
		{
			Kill:                    demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
			IncludeVictim:           false,
			EnableVoice:             false,
			EnableSpecShowXray:      false,
			HasVoiceOverride:        true,
			HasSpecShowXrayOverride: true,
		},
		{
			Kill:               demo.ClipKill{ID: "k2", Tick: 400, KillerSlot: 8},
			IncludeVictim:      false,
			EnableVoice:        true,
			EnableSpecShowXray: true,
		},
	}, BuildOptions{
		TickRate:                  64,
		KillerPreSeconds:          1,
		KillerPostSeconds:         1,
		RecordFPS:                 60,
		VideoPreset:               "n1",
		RecordOutputDir:           `D:/clips/output`,
		EnableSpecShowXray:        true,
		ForcePerPassVoiceCommands: true,
		ForcePerPassXrayCommands:  true,
		ActionSettings: ActionSettings{
			EnableVoiceIndices:  true,
			EnableVoiceIndicesH: true,
			VoiceIndicesValue:   -1,
			VoiceIndicesHValue:  -1,
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.Sequences) != 2 {
		t.Fatalf("sequence len=%d want 2", len(result.Sequences))
	}
	actions := result.Sequences[1].Actions
	assertHasAction(t, actions, "tv_listen_voice_indices 0")
	assertHasAction(t, actions, "tv_listen_voice_indices_h 0")
	assertHasAction(t, actions, "spec_show_xray -1")
	assertHasAction(t, actions, "tv_listen_voice_indices -1")
	assertHasAction(t, actions, "tv_listen_voice_indices_h -1")
	assertHasAction(t, actions, "spec_show_xray 0")
}

func TestBuild_TakePlansContainVictimView(t *testing.T) {
	result, err := Build([]Item{
		{Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7, VictimSlot: 11}, IncludeVictim: true},
	}, BuildOptions{
		TickRate:           64,
		KillerPreSeconds:   1,
		KillerPostSeconds:  1,
		VictimPreSeconds:   1,
		VictimPostSeconds:  1,
		RecordFPS:          60,
		VideoPreset:        "n1",
		RecordOutputDir:    `D:/clips/output`,
		EnableSpecShowXray: true,
		ActionSettings:     ActionSettings{VoiceIndicesValue: 0, VoiceIndicesHValue: 0},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.TakePlans) != 2 {
		t.Fatalf("take plan len=%d want 2", len(result.TakePlans))
	}
	if result.TakePlans[0].View != "killer" || result.TakePlans[1].View != "victim" {
		t.Fatalf("unexpected views: %+v", result.TakePlans)
	}
	if result.TakePlans[0].SpecMode != 1 || result.TakePlans[1].SpecMode != 1 {
		t.Fatalf("unexpected spec modes: %+v", result.TakePlans)
	}
}

func TestBuild_LastSequenceEndsWithDisconnect(t *testing.T) {
	result, err := Build([]Item{
		{Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7, VictimSlot: 11}, IncludeVictim: true},
	}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		VictimPreSeconds:  1,
		VictimPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "n1",
		RecordOutputDir:   `D:/clips/output`,
		ActionSettings:    ActionSettings{VoiceIndicesValue: 0, VoiceIndicesHValue: 0},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.Sequences) < 2 {
		t.Fatalf("sequence len=%d want >=2", len(result.Sequences))
	}
	last := result.Sequences[len(result.Sequences)-1].Actions
	assertHasAction(t, last, "disconnect")
	assertNoAction(t, last, "quit")
}

func TestBuild_InvalidPresetReturnsError(t *testing.T) {
	_, err := Build([]Item{{
		Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
	}}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "bad",
		RecordOutputDir:   `D:/clips/output`,
	})
	if err == nil || !strings.Contains(err.Error(), "video_preset") {
		t.Fatalf("expected invalid preset error, got: %v", err)
	}
}

func TestBuild_SupportsAMDPreset(t *testing.T) {
	result, err := Build([]Item{{
		Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
	}}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "a1",
		RecordOutputDir:   `D:/clips/output`,
	})
	if err != nil {
		t.Fatalf("Build with a1: %v", err)
	}
	bootstrap := result.Sequences[0].Actions
	assertHasPrefixAction(t, bootstrap, "mirv_streams settings add ffmpeg a1 ")
}

func TestBuild_AppliesRecordQualityToHardwarePreset(t *testing.T) {
	result, err := Build([]Item{{
		Kill: demo.ClipKill{ID: "k1", Tick: 200, KillerSlot: 7},
	}}, BuildOptions{
		TickRate:          64,
		KillerPreSeconds:  1,
		KillerPostSeconds: 1,
		RecordFPS:         60,
		VideoPreset:       "n1",
		RecordQuality:     "ultra",
		RecordOutputDir:   `D:/clips/output`,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	bootstrap := result.Sequences[0].Actions
	assertHasPrefixAction(t, bootstrap, "mirv_streams settings add ffmpeg n1 ")
	assertHasActionContaining(t, bootstrap, "-c:v hevc_nvenc")
	assertHasActionContaining(t, bootstrap, "-qp 10")
}

func assertHasAction(t *testing.T, actions []Action, cmd string) {
	t.Helper()
	for _, action := range actions {
		if action.Cmd == cmd {
			return
		}
	}
	t.Fatalf("action %q not found in %#v", cmd, actions)
}

func assertNoAction(t *testing.T, actions []Action, cmd string) {
	t.Helper()
	for _, action := range actions {
		if action.Cmd == cmd {
			t.Fatalf("unexpected action %q in %#v", cmd, actions)
		}
	}
}

func assertHasPrefixAction(t *testing.T, actions []Action, prefix string) {
	t.Helper()
	for _, action := range actions {
		if strings.HasPrefix(action.Cmd, prefix) {
			return
		}
	}
	t.Fatalf("action prefix %q not found in %#v", prefix, actions)
}

func assertHasActionContaining(t *testing.T, actions []Action, needle string) {
	t.Helper()
	for _, action := range actions {
		if strings.Contains(action.Cmd, needle) {
			return
		}
	}
	t.Fatalf("action containing %q not found in %#v", needle, actions)
}

func takeName(index int) string {
	return fmt.Sprintf("take%04d", index-1)
}
