package envsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cs2-highlight-tool-v2/internal/config"
)

func TestResolveCS2Exe(t *testing.T) {
	dir := t.TempDir()
	cs2 := filepath.Join(dir, "cs2.exe")
	if err := os.WriteFile(cs2, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveCS2Exe(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != cs2 {
		t.Fatalf("resolveCS2Exe = %q, want %q", got, cs2)
	}
}

func TestEnsureCS2Path_UsesConfiguredPathWithoutAutoDetect(t *testing.T) {
	exeDir := t.TempDir()
	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	cs2Path := filepath.Join(exeDir, "game", "bin", "win64", "cs2.exe")
	if err := os.MkdirAll(filepath.Dir(cs2Path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cs2Path, []byte("cs2"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.persistConfig(func(next *config.Config) error {
		next.CS2Dir = exeDir
		next.CS2Exe = ""
		return nil
	}); err != nil {
		t.Fatalf("persist config: %v", err)
	}

	oldDetect := detectCS2ExeFromSteamFn
	detectCalls := 0
	detectCS2ExeFromSteamFn = func() (string, error) {
		detectCalls++
		return "", fmt.Errorf("should not call auto detect")
	}
	t.Cleanup(func() {
		detectCS2ExeFromSteamFn = oldDetect
	})

	if err := svc.ensureCS2Path(); err != nil {
		t.Fatalf("ensureCS2Path: %v", err)
	}
	if detectCalls != 0 {
		t.Fatalf("auto detect called %d times, want 0", detectCalls)
	}
	step := findStepByID(svc.GetStartupState().Steps, componentCS2)
	if step == nil {
		t.Fatal("cs2 step missing")
	}
	if step.Status != statusReady {
		t.Fatalf("cs2 step status = %q, want %q", step.Status, statusReady)
	}
}

func TestEnsureCS2Path_AutoDetectWhenConfiguredPathInvalid(t *testing.T) {
	exeDir := t.TempDir()
	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	detected := filepath.Join(exeDir, "auto", "game", "bin", "win64", "cs2.exe")
	if err := os.MkdirAll(filepath.Dir(detected), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(detected, []byte("cs2"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.persistConfig(func(next *config.Config) error {
		next.CS2Dir = filepath.Join(exeDir, "missing")
		next.CS2Exe = ""
		return nil
	}); err != nil {
		t.Fatalf("persist config: %v", err)
	}

	oldDetect := detectCS2ExeFromSteamFn
	detectCalls := 0
	detectCS2ExeFromSteamFn = func() (string, error) {
		detectCalls++
		return detected, nil
	}
	t.Cleanup(func() {
		detectCS2ExeFromSteamFn = oldDetect
	})

	if err := svc.ensureCS2Path(); err != nil {
		t.Fatalf("ensureCS2Path: %v", err)
	}
	if detectCalls != 1 {
		t.Fatalf("auto detect called %d times, want 1", detectCalls)
	}
	state := svc.GetStartupState()
	step := findStepByID(state.Steps, componentCS2)
	if step == nil {
		t.Fatal("cs2 step missing")
	}
	if step.Status != statusReady {
		t.Fatalf("cs2 step status = %q, want %q", step.Status, statusReady)
	}
	if step.Path != detected {
		t.Fatalf("cs2 step path = %q, want %q", step.Path, detected)
	}
}

func TestEnsureCS2Path_AutoDetectFailKeepsNeedsAction(t *testing.T) {
	exeDir := t.TempDir()
	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	if _, err := svc.persistConfig(func(next *config.Config) error {
		next.CS2Dir = filepath.Join(exeDir, "missing")
		next.CS2Exe = ""
		return nil
	}); err != nil {
		t.Fatalf("persist config: %v", err)
	}

	oldDetect := detectCS2ExeFromSteamFn
	detectCS2ExeFromSteamFn = func() (string, error) {
		return "", fmt.Errorf("no steam")
	}
	t.Cleanup(func() {
		detectCS2ExeFromSteamFn = oldDetect
	})

	if err := svc.ensureCS2Path(); err != nil {
		t.Fatalf("ensureCS2Path: %v", err)
	}
	step := findStepByID(svc.GetStartupState().Steps, componentCS2)
	if step == nil {
		t.Fatal("cs2 step missing")
	}
	if step.Status != statusNeedsAction {
		t.Fatalf("cs2 step status = %q, want %q", step.Status, statusNeedsAction)
	}
}
