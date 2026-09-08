package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"strings"
	"testing"
)

func TestAuthoredQuestAssembly(t *testing.T) {
	c := authoredFixture()
	c.Quests.Quests = []content.Quest{{ID: "q", Name: "Demo", Enabled: true, AutoStart: true, Steps: []content.QuestStep{{ID: "s", Text: "Take item", Kind: "carry", Target: content.CellRef{Kind: "world_item", ID: al}}}}}
	r := AssembleAuthored(c)
	if !r.Ready() {
		t.Fatal(r.DiagnosticText())
	}
	e := engine.New(r.World)
	e.Execute("take exterior item")
	if r.World.QuestStates()[0].Status != "completed" {
		t.Fatal(r.World.QuestText(true))
	}
	c.Quests.Quests[0].Steps[0].Target.ID = "missing"
	if r = AssembleAuthored(c); r.Ready() || !strings.Contains(r.DiagnosticText(), "existing item") {
		t.Fatal("enabled missing target accepted", r.DiagnosticText())
	}
	c.Quests.Quests[0].Enabled = false
	if r = AssembleAuthored(c); !r.Ready() || len(r.World.Quests) != 0 {
		t.Fatal("draft blocked play")
	}
	c.Quests.Quests[0].Enabled = true
	c.Quests.Quests[0].Steps[0].Target.ID = af
	if r = AssembleAuthored(c); r.Ready() || !strings.Contains(r.DiagnosticText(), "takeable") {
		t.Fatal("fixed target accepted")
	}
}
