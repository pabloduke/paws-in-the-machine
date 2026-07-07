package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestHackingScreenSwapsAndRestores(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	before := mod.View()
	if !strings.Contains(before, "THE CITY") {
		t.Fatalf("expected normal room UI before login")
	}

	mod = typeLine(mod, "log in") // rewrite -> "use deck" -> shell
	mm := mod.(Model)
	if mm.shell == nil {
		t.Fatalf("'log in' should open the hacking terminal")
	}
	view := mod.View()
	if !strings.Contains(view, "CYBERDECK // DECK") {
		t.Fatalf("terminal title missing: %q", view)
	}
	for _, section := range []string{"OBJECTIVE", "LOCATION", "DISCOVERIES", "STATUS"} {
		if !strings.Contains(view, section) {
			t.Fatalf("terminal right panel should show %s: %q", section, view)
		}
	}
	if strings.Contains(view, "THE CITY") || strings.Contains(view, "LOG") {
		t.Fatalf("normal panels should be hidden while in the terminal")
	}

	// Output goes to the terminal scrollback, not the LOG.
	logLen := len(mm.entries)
	mod = typeLine(mod, "ls")
	mm = mod.(Model)
	joined := strings.Join(mm.shellEntries, "\n")
	if !strings.Contains(joined, "notes/") {
		t.Fatalf("ls output should land in shell scrollback: %q", joined)
	}
	if len(mm.entries) != logLen {
		t.Fatalf("shell commands must not touch the LOG")
	}

	// Esc closes the terminal outright.
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mm = mod.(Model)
	if mm.shell != nil {
		t.Fatalf("esc should close the terminal")
	}
	if after := mod.View(); !strings.Contains(after, "THE CITY") {
		t.Fatalf("room UI should be restored after closing the terminal")
	}
}

func TestTerminalExitReturnsToRoom(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "exit") // at the deck, exit closes the terminal
	if mod.(Model).shell != nil {
		t.Fatalf("'exit' at the deck should close the terminal")
	}
	if !strings.Contains(mod.View(), "THE CITY") {
		t.Fatalf("room UI should be restored after exit")
	}
}

func TestTerminalPromptInsidePanelAndScrollbackBottomAnchored(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mm := mod.(Model)
	if mm.shellInput.TextStyle.GetForeground() != termPromptStyle.GetForeground() {
		t.Fatalf("typed terminal text should use terminal prompt style")
	}

	scrollLines := strings.Split(stripANSI(mm.shellVP.View()), "\n")
	if len(scrollLines) < 3 {
		t.Fatalf("expected a multi-line terminal viewport, got %q", mm.shellVP.View())
	}
	if strings.TrimSpace(scrollLines[0]) != "" {
		t.Fatalf("short scrollback should be bottom-anchored, first viewport line=%q", scrollLines[0])
	}
	if !strings.Contains(scrollLines[len(scrollLines)-1], "PAWS/OS") {
		t.Fatalf("newest short scrollback should sit at the bottom, viewport=%q", strings.Join(scrollLines, "\n"))
	}

	for _, r := range "ls" {
		mod, _ = mod.Update(kr(r))
	}
	view := stripANSI(mod.View())
	lines := strings.Split(view, "\n")
	if !strings.Contains(view, "paws_in_the_machine@deck:~ $ ls") {
		t.Fatalf("typed command should render in terminal prompt:\n%s", view)
	}
	if strings.Contains(lines[len(lines)-1], "paws_in_the_machine@deck:~ $ ls") {
		t.Fatalf("terminal prompt should not render below the panel:\n%s", view)
	}
}

func TestTerminalScreenRowsAlignToWindowWidth(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	assertTerminalScreenSize(t, mod, 100, 30)
}

func TestTerminalScreenRowsAlignWithReaderOpen(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "cat ~/notes/notes.md")
	mm := mod.(Model)
	termW, readerW, _ := mm.shellDims()
	if termW != 60 || readerW != 40 {
		t.Fatalf("reader split term=%d reader=%d, want 60/40", termW, readerW)
	}
	assertTerminalScreenSize(t, mod, 100, 30)
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	assertTerminalScreenSize(t, mod, 32, 10)
}

func assertTerminalScreenSize(t *testing.T, mod tea.Model, wantW, wantH int) {
	t.Helper()
	view := mod.View()
	lines := strings.Split(view, "\n")
	if got := len(lines); got != wantH {
		t.Fatalf("terminal height=%d, want %d:\n%s", got, wantH, stripANSI(view))
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != wantW {
			t.Fatalf("terminal row %d width=%d, want %d:\n%s", i, got, wantW, stripANSI(view))
		}
	}
}

func TestMarkdownCatOpensReaderPanel(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "cat ~/notes/notes.md")
	mm := mod.(Model)
	joined := stripANSI(strings.Join(mm.shellEntries, "\n"))
	if !strings.Contains(joined, "opened ~/notes/notes.md in reader") {
		t.Fatalf("terminal should show reader notice: %q", joined)
	}
	if strings.Contains(joined, "Nothing solid yet") {
		t.Fatalf("terminal should not duplicate reader document: %q", joined)
	}
	if reader := readerText(mod); !strings.Contains(reader, "Nothing solid yet") {
		t.Fatalf("reader should render notes: %q", reader)
	}
	if mm.readerTitle != "~/notes/notes.md" || mm.readerMD == "" {
		t.Fatalf("reader metadata missing: title=%q markdown=%q", mm.readerTitle, mm.readerMD)
	}
}

func TestTextCatOpensReaderPanel(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "touch myNotes.txt")
	mod = typeLine(mod, "edit myNotes.txt")
	mod = typeRunes(mod, "plain note")
	mod, _ = mod.Update(spec(tea.KeyTab))
	mod = typeLine(mod, "cat myNotes.txt")
	mm := mod.(Model)
	joined := stripANSI(strings.Join(mm.shellEntries, "\n"))
	if !strings.Contains(joined, "opened ~/myNotes.txt in reader") {
		t.Fatalf("terminal should show txt reader notice: %q", joined)
	}
	if strings.Contains(joined, "plain note") {
		t.Fatalf("terminal should not duplicate txt reader document: %q", joined)
	}
	if reader := readerText(mod); !strings.Contains(reader, "plain note") {
		t.Fatalf("reader should render txt: %q", reader)
	}
	if mm.readerText != "plain note" || mm.readerMD != "" {
		t.Fatalf("reader text metadata mismatch text=%q md=%q", mm.readerText, mm.readerMD)
	}
}

func TestNonMarkdownCatStaysInTerminalScrollback(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh sunfarm.arc")
	mod = typeLine(mod, "cat /var/log/burial.log")
	mm := mod.(Model)
	if mm.readerMD != "" {
		t.Fatalf("non-markdown cat should not open reader: %q", mm.readerMD)
	}
	if joined := stripANSI(strings.Join(mm.shellEntries, "\n")); !strings.Contains(joined, "buried the sun") {
		t.Fatalf("non-markdown cat output should stay in terminal: %q", joined)
	}
}

func TestTerminalEditorAutosavesOnTab(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "touch myNotes.md")
	mod = typeLine(mod, "edit myNotes.md")
	mm := mod.(Model)
	if mm.editorPath != "~/myNotes.md" || !mm.readerFocus {
		t.Fatalf("edit should focus editor, path=%q focus=%v", mm.editorPath, mm.readerFocus)
	}
	mod = typeRunes(mod, "# Mine")
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mod = typeRunes(mod, "hello")
	mod, _ = mod.Update(spec(tea.KeyTab))
	mm = mod.(Model)
	if mm.editorPath != "" || mm.readerFocus {
		t.Fatalf("tab should save and return focus, path=%q focus=%v", mm.editorPath, mm.readerFocus)
	}
	if joined := stripANSI(strings.Join(mm.shellEntries, "\n")); !strings.Contains(joined, "saved ~/myNotes.md") {
		t.Fatalf("save notice missing: %q", joined)
	}
	mod = typeLine(mod, "cat myNotes.md")
	if reader := readerText(mod); !strings.Contains(reader, "// MINE") || !strings.Contains(reader, "hello") {
		t.Fatalf("saved markdown should render in reader: %q", reader)
	}
}

func TestTerminalEditorSavesOnEscBeforeClosing(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "touch scratch.txt")
	mod = typeLine(mod, "edit scratch.txt")
	mod = typeRunes(mod, "saved on close")
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if mod.(Model).shell != nil {
		t.Fatalf("esc should close terminal")
	}
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "cat scratch.txt")
	if reader := readerText(mod); !strings.Contains(reader, "saved on close") {
		t.Fatalf("esc should save editor before closing: %q", reader)
	}
}

func TestTerminalReaderFocusScrollsWithArrows(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "cat ~/notes/notes.md")
	mod, _ = mod.Update(spec(tea.KeyTab))
	if !mod.(Model).readerFocus {
		t.Fatalf("tab should focus reader")
	}
	mod, _ = mod.Update(kr('x'))
	if got := mod.(Model).shellInput.Value(); got != "" {
		t.Fatalf("typing while reader is focused should not reach prompt: %q", got)
	}
	before := mod.(Model).shellReader.YOffset
	mod, _ = mod.Update(spec(tea.KeyDown))
	if after := mod.(Model).shellReader.YOffset; after < before {
		t.Fatalf("reader down should not move upward: before=%d after=%d", before, after)
	}
	mod, _ = mod.Update(spec(tea.KeyTab))
	if mod.(Model).readerFocus {
		t.Fatalf("second tab should return focus to terminal")
	}
	mod, _ = mod.Update(kr('x'))
	if got := mod.(Model).shellInput.Value(); got != "x" {
		t.Fatalf("typing after returning focus should reach prompt: %q", got)
	}
}

func TestDeckLoginFromPanel(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	// The deck is carried, in a green UPLINK section of the left panel,
	// separated from the hubs.
	if view := mod.View(); !strings.Contains(view, "UPLINK") || !strings.Contains(view, "the deck") {
		t.Fatalf("deck should be listed in the left panel while in the lair:\n%s", view)
	}

	// Focus the panel, arrow to the deck (last item), Enter logs in.
	mod, _ = mod.Update(spec(tea.KeyTab))
	items := mod.(Model).panelItems()
	if items[len(items)-1].kind != panelDeck {
		t.Fatalf("deck should be the last focusable panel item")
	}
	for mod.(Model).selected != len(items)-1 {
		mod, _ = mod.Update(spec(tea.KeyDown))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if mod.(Model).shell == nil {
		t.Fatalf("Enter on the deck row should log in")
	}
}

// The deck lives in the lair and nowhere else (user ruling
// 2026-07-07, reversing the earlier carried-deck behavior): hacking
// starts at home, or there's no point to having a lair.
func TestDeckOnlyReachableInLair(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "north") // leave the lair for the coffee shop
	for _, it := range mod.(Model).panelItems() {
		if it.kind == panelDeck {
			t.Fatalf("no deck panel item should exist outside the lair")
		}
	}
	mod = typeLine(mod, "use deck")
	if mod.(Model).shell != nil {
		t.Fatalf("the deck should not open outside the lair")
	}

	mod = typeLine(mod, "south") // home again
	found := false
	for _, it := range mod.(Model).panelItems() {
		if it.kind == panelDeck {
			found = true
		}
	}
	if !found {
		t.Fatalf("the deck should have a panel item in the lair")
	}
	mod = typeLine(mod, "use deck")
	if mod.(Model).shell == nil {
		t.Fatalf("the deck should open the terminal in the lair")
	}
}

func TestHackingBeatSetsFlags(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh sunfarm.arc")
	mod = typeLine(mod, "cat /var/log/burial.log")
	if !w.Flags["heard_whisper"] {
		t.Fatalf("reading burial.log should set heard_whisper; flags=%v", w.Flags)
	}
	mod = typeLine(mod, "cp /srv/archive/sun.frag ~/")
	if !w.Flags["got_sun_fragment"] {
		t.Fatalf("downloading sun.frag should set got_sun_fragment; flags=%v", w.Flags)
	}
	mod = typeLine(mod, "cat ~/notes/notes.md")
	if reader := readerText(mod); !strings.Contains(reader, "fragment") {
		t.Fatalf("notes should mention the copied fragment: %q", reader)
	}
}

func TestMicroslopPasswordPuzzle(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh microslop")
	if joined := strings.Join(mod.(Model).shellEntries, "\n"); !strings.Contains(joined, "Network is unreachable") {
		t.Fatalf("ssh microslop should require a local route first: %q", joined)
	}
	if mod.(Model).shell.HostName() != "deck" {
		t.Fatalf("missing route should not connect")
	}
	mod = typeLine(mod, "exit")

	mod = typeLine(mod, "north")
	mod = typeLine(mod, "talk barista")
	mod, _ = mod.Update(spec(tea.KeyEnter)) // purr
	mod, _ = mod.Update(spec(tea.KeyEnter)) // accept tribute
	mod = typeLine(mod, "talk barista")
	mod = chooseDialogueContaining(t, mod, "Microslop")
	if !w.Flags["knows_microslop_password"] {
		t.Fatalf("barista should reveal Microslop password; flags=%v", w.Flags)
	}

	mod = typeLine(mod, "parkour hound")
	mod = spendPendingLevelUp(mod)
	mod = typeLine(mod, "use rack")
	if !w.Flags["microslop_route_open"] {
		t.Fatalf("using the backroom rack should open the Microslop route; flags=%v", w.Flags)
	}

	// The deck lives in the lair (deck_test.go): the route is open,
	// but hacking it means going home first.
	mod = typeLine(mod, "north") // back room -> coffee shop
	mod = typeLine(mod, "south") // coffee shop -> lair
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh microslop")
	if view := mod.View(); !strings.Contains(view, "password:") {
		t.Fatalf("ssh microslop should show password prompt after route opens: %q", view)
	}
	mod = typeLine(mod, "wrong")
	if mod.(Model).shell.HostName() != "deck" {
		t.Fatalf("wrong password should stay on deck")
	}
	if joined := strings.Join(mod.(Model).shellEntries, "\n"); !strings.Contains(joined, "Permission denied") {
		t.Fatalf("wrong password should log refusal: %q", joined)
	}

	mod = typeLine(mod, "ssh microslop")
	mod = typeLine(mod, "apple")
	if mod.(Model).shell.HostName() != "microslop" {
		t.Fatalf("correct password should connect to microslop")
	}
	if view := mod.View(); !strings.Contains(view, "CYBERDECK // MICROSLOP") {
		t.Fatalf("microslop title missing: %q", view)
	}
	mod = typeLine(mod, "grep sun /var/log/access.log")
	mod = typeLine(mod, "cat /srv/archive/sun_notice.txt")
	if !w.Flags["read_microslop_sun_notice"] {
		t.Fatalf("reading Microslop notice should set flag; flags=%v", w.Flags)
	}
	mod = typeLine(mod, "cp /srv/archive/sun_notice.txt ~/")
	if !w.Flags["got_microslop_notice"] {
		t.Fatalf("copying Microslop notice should set flag; flags=%v", w.Flags)
	}
	mod = typeLine(mod, "cat ~/notes/notes.md")
	if reader := readerText(mod); !strings.Contains(reader, "apple") ||
		!strings.Contains(reader, "SUNFARM-ARC") ||
		!strings.Contains(reader, "sunlight") || !strings.Contains(reader, "liability notice") {
		t.Fatalf("notes should record Microslop discoveries: %q", reader)
	}
}

func readerText(mod tea.Model) string {
	return stripANSI(mod.(Model).shellReader.View())
}

func typeRunes(mod tea.Model, text string) tea.Model {
	for _, r := range text {
		mod, _ = mod.Update(kr(r))
	}
	return mod
}

func chooseDialogueContaining(t *testing.T, mod tea.Model, text string) tea.Model {
	t.Helper()
	mm := mod.(Model)
	if mm.dialogue == nil {
		t.Fatalf("expected active dialogue")
	}
	idx := -1
	for i, opt := range mm.dialogue.Options() {
		if strings.Contains(opt.Choice.Text, text) {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("dialogue option containing %q not found", text)
	}
	for mod.(Model).dlgSel != idx {
		mod, _ = mod.Update(spec(tea.KeyDown))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	return mod
}

func spendPendingLevelUp(mod tea.Model) tea.Model {
	if mod.(Model).modal == modalLevelUp {
		mod, _ = mod.Update(spec(tea.KeyEnter))
	}
	return mod
}

func TestHackingScreenNarrowTerminal(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	// Must render without panicking and keep the terminal present.
	if view := mod.View(); !strings.Contains(view, "CYBERDECK // DEC") {
		t.Fatalf("narrow render lost the terminal: %q", view)
	}
}
