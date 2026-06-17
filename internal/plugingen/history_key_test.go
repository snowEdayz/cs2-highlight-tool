package plugingen

import "testing"

func TestBuildProduceHistoryKeyWithSourceIDDistinguishesFullRoundSegments(t *testing.T) {
	keyA := BuildProduceHistoryKeyWithSourceID("demo.dem", "full_round_pov", 1, nil, "full_round_pov:r1:p765")
	keyB := BuildProduceHistoryKeyWithSourceID("demo.dem", "full_round_pov", 1, nil, "full_round_pov:r2:p765")

	if keyA == keyB {
		t.Fatalf("full-round source ids should produce distinct keys: %q", keyA)
	}
}

func TestBuildProduceHistoryKeyWithSourceIDKeepsClipKeysCompatible(t *testing.T) {
	legacy := BuildProduceHistoryKey("demo.dem", "victim", 1, []string{"k2", "k1"})
	next := BuildProduceHistoryKeyWithSourceID("demo.dem", "victim", 1, []string{"k1", "k2"}, "")

	if next != legacy {
		t.Fatalf("clip key changed: got %q want %q", next, legacy)
	}
}
