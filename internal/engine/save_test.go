package engine

import "testing"

// saveWorld builds a small world twice — the "same game, fresh
// launch" situation load has to handle.
func saveWorld() *World {
	w := NewWorld()
	home := NewEntity("home", "Home")
	away := NewEntity("away", "Away")
	coin := NewEntity("coin", "a coin").With(Portable{})
	npc := NewEntity("npc", "the npc").With(Placed{Fn: func(w *World) string {
		if w.Flags["moved"] {
			return "away"
		}
		return "home"
	}})
	w.Root.Add(home, away)
	home.Add(w.Player, coin, npc)
	return w
}

func TestSaveRoundTrip(t *testing.T) {
	w := saveWorld()
	// Play a little: take the coin, walk away, flip flags, spend XP.
	w.Player.Add(w.FindID("coin"))
	w.FindID("away").Add(w.Player)
	w.Flags["moved"] = true
	w.Flags["heard_whisper"] = true
	w.Stats.Stealth = 12
	w.XP, w.Level, w.StatPoints = 3, 2, 1
	w.CheckEvents() // npc derives to away

	data, err := Snapshot(w).Marshal()
	if err != nil {
		t.Fatal(err)
	}

	// Fresh launch.
	w2 := saveWorld()
	s, err := UnmarshalSave(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(w2); err != nil {
		t.Fatal(err)
	}

	if w2.Room().ID != "away" {
		t.Fatalf("player position lost: %s", w2.Room().ID)
	}
	if !w2.Carried(w2.FindID("coin")) {
		t.Fatalf("inventory lost")
	}
	if !w2.Flags["heard_whisper"] || !w2.Flags["moved"] {
		t.Fatalf("flags lost: %v", w2.Flags)
	}
	if w2.Stats.Stealth != 12 || w2.XP != 3 || w2.Level != 2 || w2.StatPoints != 1 {
		t.Fatalf("numbers lost: %+v xp=%d lv=%d pts=%d", w2.Stats, w2.XP, w2.Level, w2.StatPoints)
	}
	if w2.Seed != w.Seed {
		t.Fatalf("seed must survive — no save-scumming checks")
	}
	if w2.FindID("npc").Parent.ID != "away" {
		t.Fatalf("placed entity should re-derive from loaded flags: %s",
			w2.FindID("npc").Parent.ID)
	}
}

func TestLoadIsNotAnArrival(t *testing.T) {
	w := saveWorld()
	w.FindID("away").Add(w.Player)
	data, _ := Snapshot(w).Marshal()

	w2 := saveWorld()
	w2.Rules = []When{{
		Enter: "away",
		Do:    func(*World) string { return "you arrive" },
	}}
	s, _ := UnmarshalSave(data)
	if err := s.Apply(w2); err != nil {
		t.Fatal(err)
	}
	w2.CheckEvents() // first action after load, still in away
	if len(w2.Pending) != 0 {
		t.Fatalf("loading must not fire Enter rules: %v", w2.Pending)
	}
}

func TestApplyOntoDirtyWorld(t *testing.T) {
	w := saveWorld()
	data, _ := Snapshot(w).Marshal() // clean start: player home, no flags

	// Play the same world into a mess, then load the clean save back.
	w.Player.Add(w.FindID("coin"))
	w.FindID("away").Add(w.Player)
	w.Flags["moved"] = true
	w.CheckEvents()

	s, _ := UnmarshalSave(data)
	if err := s.Apply(w); err != nil {
		t.Fatal(err)
	}
	if w.Room().ID != "home" || w.Flags["moved"] || w.Carried(w.FindID("coin")) {
		t.Fatalf("load must fully overwrite a dirty world")
	}
	if w.FindID("npc").Parent.ID != "home" {
		t.Fatalf("placed entity should re-derive after load: %s", w.FindID("npc").Parent.ID)
	}
}

func TestVersionMismatchRejected(t *testing.T) {
	s := SaveState{Version: 99}
	if err := s.Apply(saveWorld()); err == nil {
		t.Fatalf("wrong-version save must be rejected")
	}
}
