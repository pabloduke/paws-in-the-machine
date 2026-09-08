package engine

import (
	"strings"
	"testing"
)

func questWorld() *World {
	w := saveWorld()
	w.Quests = []Quest{{ID: "sample", Name: "Sample", AutoStart: true, CompletionText: "Done", Steps: []QuestStep{{ID: "take", Text: "Take the coin", Kind: "carry", Target: "coin"}, {ID: "visit", Text: "Visit away", Kind: "visit", Target: "away"}}}}
	w.EvaluateQuests()
	return w
}
func TestQuestActionsAndSave(t *testing.T) {
	w := questWorld()
	e := New(w)
	if w.QuestStates()[0].Status != "active" {
		t.Fatal("not automatically active")
	}
	e.Execute("look")
	if w.QuestStates()[0].Steps[0].Complete {
		t.Fatal("unrelated action completed objective")
	}
	e.Execute("take coin")
	if !w.QuestStates()[0].Steps[0].Complete {
		t.Fatal("real pickup did not complete objective")
	}
	e.Execute("drop coin")
	if !w.QuestStates()[0].Steps[0].Complete {
		t.Fatal("dropping undid progress")
	}
	w.FindID("away").Add(w.Player)
	w.CheckEvents()
	if w.QuestStates()[0].Status != "completed" || len(w.Pending) != 1 || w.Pending[0] != "Done" {
		t.Fatal(w.QuestText(true), w.Pending)
	}
	w.CheckEvents()
	if len(w.Pending) != 1 {
		t.Fatal("duplicate completion")
	}
	saved := Snapshot(w)
	fresh := questWorld()
	if err := saved.Apply(fresh); err != nil {
		t.Fatal(err)
	}
	fresh.CheckEvents()
	if fresh.QuestStates()[0].Status != "completed" || len(fresh.Pending) != 0 {
		t.Fatal("save lost completion or repeated announcement")
	}
}
func TestQuestOrderingAndDebugger(t *testing.T) {
	w := questWorld()
	w.FindID("away").Add(w.Player)
	w.CheckEvents()
	states := w.QuestStates()
	if states[0].Steps[1].Complete || !states[0].Steps[1].Satisfied {
		t.Fatal("out of order step completed")
	}
	before := Snapshot(w)
	if _, ok := w.DevCommand("sudo devmode --meow"); !ok || !w.DevMode {
		t.Fatal("enable failed")
	}
	if w.Flags[questFlag("sample", "step:take")] != before.Flags[questFlag("sample", "step:take")] {
		t.Fatal("enable changed progress")
	}
	if out, _ := w.DevCommand("quest-debug"); !strings.Contains(out, "Item is not in inventory") {
		t.Fatal(out)
	}
	w.DevCommand("quest-debug complete sample")
	if !w.DevModified || w.QuestStates()[0].Status != "completed" {
		t.Fatal("force failed")
	}
	w.Flags["unrelated"] = true
	w.DevCommand("quest-debug reset sample")
	if !w.Flags["unrelated"] || w.QuestStates()[0].Steps[0].Complete {
		t.Fatal("reset scope")
	}
	w.Player.Add(w.FindID("coin"))
	w.CheckEvents()
	if w.QuestStates()[0].Status != "completed" {
		t.Fatal("already satisfied steps should complete in order")
	}
	for i := 0; i < 120; i++ {
		w.traceQuest("test")
	}
	if len(w.QuestTrace) != 100 {
		t.Fatal("unbounded trace")
	}
	w.DevCommand("sudo devmode --off")
	if w.DevMode || !strings.Contains(w.DevLabel(), "MODIFIED") {
		t.Fatal("disable hid modification marker")
	}
	w.DevCommand("quest-debug reset sample")
	if w.QuestStates()[0].Status != "completed" {
		t.Fatal("disabled debugger mutated quest")
	}
	fresh := questWorld()
	if err := Snapshot(w).Apply(fresh); err != nil {
		t.Fatal(err)
	}
	if !fresh.DevModified {
		t.Fatal("saved forced progress lost marker")
	}
	if fresh.DevMode {
		t.Fatal("dev mode persisted")
	}
}
func TestManualQuestActivation(t *testing.T) {
	w := saveWorld()
	w.Quests = []Quest{{ID: "manual", Name: "Manual", Steps: []QuestStep{{ID: "s", Kind: "visit", Target: "home"}}}}
	w.CheckEvents()
	if w.QuestStates()[0].Status != "available" {
		t.Fatal("manual quest auto-started")
	}
	New(w).Execute("quests start manual")
	if w.QuestStates()[0].Status != "completed" {
		t.Fatal("manual activation failed")
	}
}
