package dialogue_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

func dialogueWorld() (*engine.World, *engine.Entity) {
	w := engine.NewWorld()
	w.Stats.Charm = 8
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	shard := engine.NewEntity("shard", "a data-shard").With(engine.Portable{})
	barista := engine.NewEntity("barista", "the barista", "barista").With(
		dialogue.Talkable{
			Start: "hello",
			Nodes: map[string]dialogue.Node{
				"hello": {
					Text: "That cat again. You lost?",
					Choices: []dialogue.Choice{
						{
							Text: "Purr like you own the place.",
							Require: []dialogue.Requirement{
								dialogue.StatAtLeast("charm", 8),
								dialogue.MissingFlag("barista_softened"),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag("barista_softened"),
								dialogue.AwardOnce("xp_barista", 5),
							},
							Next: "softened",
						},
						{
							Text:    "Show the data-shard.",
							Require: []dialogue.Requirement{dialogue.HasItem("shard")},
							Effects: []dialogue.Effect{dialogue.SetFlag("barista_saw_shard")},
							End:     true,
						},
						{Text: "Leave.", End: true},
					},
				},
				"softened": {
					Text: "Fine. One saucer. Don't make it weird.",
					Choices: []dialogue.Choice{
						{Text: "Leave.", End: true},
					},
				},
			},
		},
	)
	w.Root.Add(room)
	room.Add(barista, shard, w.Player)
	return w, barista
}

func TestDialogueChoicesAndEffects(t *testing.T) {
	w, barista := dialogueWorld()
	session, err := dialogue.Start(w, barista)
	if err != nil {
		t.Fatal(err)
	}

	rendered := session.Render()
	if !strings.Contains(rendered, "[Charm 8] Purr like you own") {
		t.Fatalf("charm choice should be visible with derived tag: %q", rendered)
	}
	if strings.Contains(rendered, "✗") {
		t.Fatalf("qualifying charm choice should not be locked: %q", rendered)
	}
	if strings.Contains(rendered, "Show the data-shard") {
		t.Fatalf("item-gated choice should be hidden before taking shard: %q", rendered)
	}

	out := session.Choose(1)
	if !w.Flags["barista_softened"] || !strings.Contains(out, "+5 XP") {
		t.Fatalf("choice should set flag and award XP: %q flags=%v", out, w.Flags)
	}
	if !strings.Contains(out, "One saucer") {
		t.Fatalf("choice should advance to softened node: %q", out)
	}
}

func TestHasItemRequirementAndEnd(t *testing.T) {
	w, barista := dialogueWorld()
	shard := w.FindID("shard")
	w.Player.Add(shard)

	session, err := dialogue.Start(w, barista)
	if err != nil {
		t.Fatal(err)
	}
	rendered := session.Render()
	if !strings.Contains(rendered, "Show the data-shard") {
		t.Fatalf("item-gated choice should be visible when carried: %q", rendered)
	}

	out := session.Choose(2)
	if out != "" {
		t.Fatalf("ending choice with no Say effect should be quiet, got %q", out)
	}
	if !session.Done() || !w.Flags["barista_saw_shard"] {
		t.Fatalf("choice should end and set flag; done=%v flags=%v", session.Done(), w.Flags)
	}
}

func TestOnceFlagHidesUsedChoice(t *testing.T) {
	w := engine.NewWorld()
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	npc := engine.NewEntity("npc", "an informant").With(
		dialogue.Talkable{
			Nodes: map[string]dialogue.Node{
				"start": {
					Text: "You get one useful answer.",
					Choices: []dialogue.Choice{
						{
							Text:     "Ask the useful question.",
							OnceFlag: "asked_useful_question",
							Next:     "start",
						},
						{Text: "Leave.", End: true},
					},
				},
			},
		},
	)
	w.Root.Add(room)
	room.Add(npc, w.Player)

	session, err := dialogue.Start(w, npc)
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Options()) != 2 {
		t.Fatalf("expected two initial choices, got %d", len(session.Options()))
	}

	out := session.Choose(1)
	if !w.Flags["asked_useful_question"] {
		t.Fatalf("once flag was not set after choice: %v", w.Flags)
	}
	if strings.Contains(out, "Ask the useful question") {
		t.Fatalf("once-only choice should be hidden after use: %q", out)
	}
	if opts := session.Options(); len(opts) != 1 || opts[0].Choice.Text != "Leave." {
		t.Fatalf("expected only Leave after once choice, got %#v", opts)
	}
}

func TestStatGateLocksAndUnlocks(t *testing.T) {
	w, barista := dialogueWorld()
	w.Stats.Charm = 5

	session, err := dialogue.Start(w, barista)
	if err != nil {
		t.Fatal(err)
	}
	rendered := session.Render()
	if !strings.Contains(rendered, "[Charm 8] Purr like you own the place. ✗") {
		t.Fatalf("under-stat choice should be visible, tagged, and locked: %q", rendered)
	}

	out := session.Choose(1)
	if !strings.Contains(out, "Your Charm isn't up to that yet. (Charm 5/8)") {
		t.Fatalf("locked choice should refuse with stat gap: %q", out)
	}
	if w.Flags["barista_softened"] || session.Done() {
		t.Fatalf("locked choice must apply nothing; flags=%v done=%v", w.Flags, session.Done())
	}

	w.Stats.Charm = 8
	if strings.Contains(session.Render(), "✗") {
		t.Fatalf("lock should re-evaluate once trained: %q", session.Render())
	}
	out = session.Choose(1)
	if !w.Flags["barista_softened"] || !strings.Contains(out, "One saucer") {
		t.Fatalf("trained stat should unlock the choice: %q flags=%v", out, w.Flags)
	}
}

func TestStartRequiresTalkable(t *testing.T) {
	w := engine.NewWorld()
	rock := engine.NewEntity("rock", "a rock")
	if _, err := dialogue.Start(w, rock); err == nil || !strings.Contains(err.Error(), "nothing to say") {
		t.Fatalf("expected no-dialogue error, got %v", err)
	}
}
