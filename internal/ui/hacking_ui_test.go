package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
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
	// The right panel idles blank except for the screen-saver art —
	// no objectives, no status (user ruling 2026-07-10).
	if !strings.Contains(view, "zzz") {
		t.Fatalf("idle right panel should show the screen-saver art: %q", view)
	}
	for _, section := range []string{"OBJECTIVE", "LOCATION", "DISCOVERIES", "STATUS"} {
		if strings.Contains(view, section) {
			t.Fatalf("idle right panel must not show %s: %q", section, view)
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
	// The newest line is the messenger's login announcement; the boot
	// banner sits just above it.
	if !strings.Contains(scrollLines[len(scrollLines)-1], "[messenger]") ||
		!strings.Contains(scrollLines[len(scrollLines)-2], "CantOS") {
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

func TestTerminalHyphenDoesNotWrapPrompt(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")

	mod, _ = mod.Update(kr('-'))
	mm := mod.(Model)
	if got := mm.shellInput.Value(); got != "-" {
		t.Fatalf("hyphen input=%q, want %q", got, "-")
	}
	if got, max := lipgloss.Width(mm.shellInput.View()), mm.shellVP.Width; got > max {
		t.Fatalf("hyphen prompt width=%d exceeds panel width=%d", got, max)
	}
}

func TestTerminalTabCompletesAndArrowsRecallHistory(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")

	mod = typeRunes(mod, "sca")
	mod, _ = mod.Update(spec(tea.KeyTab))
	if got := mod.(Model).shellInput.Value(); got != "scan " {
		t.Fatalf("tab completion=%q, want %q", got, "scan ")
	}
	mod = typeRunes(mod, "sunfarm.arc")
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mod = typeLine(mod, "pwd")

	mod = typeRunes(mod, "draft")
	mod, _ = mod.Update(spec(tea.KeyUp))
	if got := mod.(Model).shellInput.Value(); got != "pwd" {
		t.Fatalf("first up=%q, want most recent command", got)
	}
	mod, _ = mod.Update(spec(tea.KeyUp))
	if got := mod.(Model).shellInput.Value(); got != "scan sunfarm.arc" {
		t.Fatalf("second up=%q, want previous command", got)
	}
	mod, _ = mod.Update(spec(tea.KeyDown))
	mod, _ = mod.Update(spec(tea.KeyDown))
	if got := mod.(Model).shellInput.Value(); got != "draft" {
		t.Fatalf("down should restore draft, got %q", got)
	}
}

func TestTerminalMasksPasswordsAndExcludesThemFromHistory(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh jane_doe@microslop")
	if !mod.(Model).shell.AwaitingPassword() {
		t.Fatalf("ssh should await a password")
	}

	mod = typeRunes(mod, "wrong")
	if view := stripANSI(mod.View()); strings.Contains(view, "wrong") || !strings.Contains(view, "*****") {
		t.Fatalf("password input should be masked: %q", view)
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mm := mod.(Model)
	if joined := stripANSI(strings.Join(mm.shellEntries, "\n")); strings.Contains(joined, "password: wrong") {
		t.Fatalf("scrollback leaked password: %q", joined)
	}
	mod, _ = mod.Update(spec(tea.KeyUp))
	if got := mod.(Model).shellInput.Value(); got != "ssh jane_doe@microslop" {
		t.Fatalf("history should skip password, got %q", got)
	}
}

func TestPanelShortcutHintsAreVisible(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	if view := stripANSI(mod.View()); !strings.Contains(view, "shift+tab: focus") {
		t.Fatalf("overworld should show panel focus shortcut: %q", view)
	}

	mod = typeLine(mod, "use deck")
	view := stripANSI(mod.View())
	if !strings.Contains(view, "Tab: complete") || !strings.Contains(view, "Shift+Tab: panel") {
		t.Fatalf("deck should show completion and panel shortcuts: %q", view)
	}
	mod, _ = mod.Update(spec(tea.KeyShiftTab))
	if !mod.(Model).readerFocus || !strings.Contains(stripANSI(mod.View()), "Shift+Tab: terminal") {
		t.Fatalf("shift+tab should select the deck panel and show the return shortcut")
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
	mod, _ = mod.Update(spec(tea.KeyShiftTab))
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

func TestTerminalEditorAutosavesOnShiftTab(t *testing.T) {
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
	mod, _ = mod.Update(spec(tea.KeyShiftTab))
	mm = mod.(Model)
	if mm.editorPath != "" || mm.readerFocus {
		t.Fatalf("shift+tab should save and return focus, path=%q focus=%v", mm.editorPath, mm.readerFocus)
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
	mod, _ = mod.Update(spec(tea.KeyShiftTab))
	if !mod.(Model).readerFocus {
		t.Fatalf("shift+tab should focus reader")
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
	mod, _ = mod.Update(spec(tea.KeyShiftTab))
	if mod.(Model).readerFocus {
		t.Fatalf("second shift+tab should return focus to terminal")
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
	mod, _ = mod.Update(spec(tea.KeyShiftTab))
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
	mod = typeLine(mod, "scan microslop")
	if joined := strings.Join(mod.(Model).shellEntries, "\n"); !strings.Contains(joined, "22    SSH      | open") {
		t.Fatalf("Microslop should be directly reachable from the lair: %q", joined)
	}
	mod = typeLine(mod, "exit")

	mod = typeLine(mod, "north")
	mod = typeLine(mod, "meow barista")
	mod, _ = mod.Update(spec(tea.KeyEnter)) // purr
	mod, _ = mod.Update(spec(tea.KeyEnter)) // accept tribute
	mod = typeLine(mod, "examine post-it")
	if !w.Flags["read_microslop_badge"] {
		t.Fatalf("examining the post-it should reveal the Microslop password; flags=%v", w.Flags)
	}

	// The deck lives in the lair (deck_test.go), so the credential comes home.
	mod = typeLine(mod, "south")
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh jane_doe@microslop")
	if view := mod.View(); !strings.Contains(view, "password:") {
		t.Fatalf("ssh jane_doe@microslop should show password prompt after route opens: %q", view)
	}
	if got := mod.(Model).mainTerminalTheme().bright; got != termGreen {
		t.Fatalf("password prompt should retain local green theme, got %v", got)
	}
	mod = typeLine(mod, "wrong")
	if mod.(Model).shell.HostName() != "deck" {
		t.Fatalf("wrong password should stay on deck")
	}
	if joined := strings.Join(mod.(Model).shellEntries, "\n"); !strings.Contains(joined, "Permission denied") {
		t.Fatalf("wrong password should log refusal: %q", joined)
	}

	mod = typeLine(mod, "ssh jane_doe@microslop")
	mod = typeLine(mod, "apple")
	if mod.(Model).shell.HostName() != "microslop" {
		t.Fatalf("correct password should connect to microslop")
	}
	mm := mod.(Model)
	if got := mm.mainTerminalTheme().bright; got != termAmber {
		t.Fatalf("remote terminal should use amber theme, got %v", got)
	}
	if got := mm.shellInput.TextStyle.GetForeground(); got != termAmber {
		t.Fatalf("remote terminal input should be amber, got %v", got)
	}
	if got := termTitleStyle.GetForeground(); got != termGreen {
		t.Fatalf("right-side terminal panels should remain green, got %v", got)
	}
	if view := mod.View(); !strings.Contains(view, "CYBERDECK // MICROSLOP") {
		t.Fatalf("microslop title missing: %q", view)
	}
	mod = typeLine(mod, "grep -ir 1008476 /")
	if joined := strings.Join(mod.(Model).shellEntries, "\n"); !strings.Contains(joined, "/srv/hr/rif_q3.txt") {
		t.Fatalf("employee ID should locate the layoff plans: %q", joined)
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
	// The full source, not the viewport: the mission checklist above
	// the field notes pushes these below the visible fold.
	if notes := mod.(Model).readerMD; !strings.Contains(notes, "jane_doe") ||
		!strings.Contains(notes, "1008476") ||
		!strings.Contains(notes, "apple") ||
		!strings.Contains(notes, "SUNFARM-ARC") ||
		!strings.Contains(notes, "sunlight") || !strings.Contains(notes, "liability notice") {
		t.Fatalf("notes should record the badge and Microslop discoveries: %q", notes)
	}
	mod = typeLine(mod, "exit")
	mm = mod.(Model)
	if got := mm.mainTerminalTheme().bright; got != termGreen {
		t.Fatalf("returning to deck should restore green theme, got %v", got)
	}
}

// vim opens the same editor modal: normal mode swallows text, i
// inserts, esc returns to normal, :wq saves, and esc never closes the
// terminal out from under the buffer.
func TestVimEditorModalFlow(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "touch plan.md")
	mod = typeLine(mod, "vim plan.md")
	mm := mod.(Model)
	if mm.editorPath != "~/plan.md" || !mm.editorVim || mm.vimInsert {
		t.Fatalf("vim should open a modal buffer in normal mode: path=%q vim=%v insert=%v",
			mm.editorPath, mm.editorVim, mm.vimInsert)
	}
	// Normal mode: typing is not inserting.
	mod = typeRunes(mod, "junk")
	if v := mod.(Model).shellEditor.Value(); v != "" {
		t.Fatalf("normal mode must not insert text: %q", v)
	}
	// Esc in normal mode hints instead of closing the terminal.
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mm = mod.(Model)
	if mm.shell == nil || mm.editorPath == "" {
		t.Fatal("esc in normal mode must not close the buffer or terminal")
	}
	if !strings.Contains(mm.vimMsg, ":q") {
		t.Fatalf("esc in normal mode should hint :q, got %q", mm.vimMsg)
	}
	// i enters insert mode; esc leaves it.
	mod = typeRunes(mod, "i")
	if !mod.(Model).vimInsert {
		t.Fatal("i should enter insert mode")
	}
	mod = typeRunes(mod, "nine lives")
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mm = mod.(Model)
	if mm.vimInsert || mm.shellEditor.Value() != "nine lives" {
		t.Fatalf("esc should return to normal with text kept: insert=%v value=%q",
			mm.vimInsert, mm.shellEditor.Value())
	}
	// :wq saves and closes.
	mod = typeRunes(mod, ":wq")
	if cmd := mod.(Model).vimCmd; cmd != ":wq" {
		t.Fatalf("pending command line should show :wq, got %q", cmd)
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mm = mod.(Model)
	if mm.editorPath != "" || mm.editorVim {
		t.Fatalf(":wq should close the buffer: path=%q vim=%v", mm.editorPath, mm.editorVim)
	}
	mod = typeLine(mod, "cat plan.md")
	if reader := readerText(mod); !strings.Contains(reader, "nine lives") {
		t.Fatalf(":wq should have saved the file: %q", reader)
	}
}

// :q abandons the buffer: the file keeps its saved content.
func TestVimQuitDiscardsChanges(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "touch plan.md")
	mod = typeLine(mod, "vim plan.md")
	mod = typeRunes(mod, "i")
	mod = typeRunes(mod, "doomed draft")
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mod = typeRunes(mod, ":q")
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mm := mod.(Model)
	if mm.editorPath != "" {
		t.Fatalf(":q should close the buffer, path=%q", mm.editorPath)
	}
	mod = typeLine(mod, "cat plan.md")
	if reader := readerText(mod); strings.Contains(reader, "doomed draft") {
		t.Fatalf(":q must not save: %q", reader)
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

// The messenger holds the modal right panel: login announces waiting
// messages, `messenger` opens the thread (reading it), a mid-session
// flag delivers a new message with a scrollback notice, and the reader
// takes the panel back when a document opens.
func TestMessengerPanelFlow(t *testing.T) {
	w := game.NewWorld()
	mod := typeLine(newSized(w), "use deck")
	mm := mod.(Model)

	// The welcome brief is waiting: announced in the scrollback (the
	// idle right panel shows nothing but the screen-saver art).
	if joined := strings.Join(mm.shellEntries, "\n"); !strings.Contains(joined, "[messenger] 1 unread") {
		t.Fatalf("login should announce the waiting message: %q", joined)
	}

	// Open the panel: contact title, message text, thread marked read.
	mod = typeLine(mod, "messenger")
	mm = mod.(Model)
	if !mm.msgOpen {
		t.Fatalf("the messenger command should open the panel")
	}
	v := mod.View()
	if !strings.Contains(v, "MESSENGER // MARDUK") || !strings.Contains(v, "barista") {
		t.Fatalf("the panel should show the contact and the welcome brief: %q", v)
	}
	if mm.deckCfg.Messenger.Unread(w) != 0 {
		t.Fatalf("opening the panel should read the thread")
	}

	// Toggle closed: the panel idles again (screen-saver art).
	mod = typeLine(mod, "messenger")
	if v := mod.View(); strings.Contains(v, "MESSENGER //") || !strings.Contains(v, "zzz") {
		t.Fatalf("the second messenger should hand the panel back to idle: %q", v)
	}

	// A flag set mid-session delivers a message at once — no timers,
	// no logout needed — with a dim notice in the scrollback.
	mod = typeLine(mod, "ssh sunfarm.arc")
	mod = typeLine(mod, "cat /var/log/burial.log") // sets heard_whisper
	mm = mod.(Model)
	if joined := strings.Join(mm.shellEntries, "\n"); !strings.Contains(joined, "[messenger] 1 unread") {
		t.Fatalf("the new arrival should be announced: %q", joined)
	}

	// The reader is modal over the messenger: opening a document while
	// the thread is up takes the panel and closes it.
	mod = typeLine(mod, "messenger")
	mod = typeLine(mod, "cat ~/notes/notes.md")
	mm = mod.(Model)
	if mm.msgOpen {
		t.Fatalf("opening a document should close the messenger")
	}
	if v := mod.View(); !strings.Contains(v, "READER //") {
		t.Fatalf("the reader should hold the panel: %q", v)
	}
}

// A deck with no messenger service refuses the command gently.
func TestMessengerWithoutService(t *testing.T) {
	w := game.NewWorld()
	mod := typeLine(newSized(w), "use deck")
	mm := mod.(Model)
	mm.deckCfg.Messenger = hacking.Messenger{}
	var tm tea.Model = mm
	tm = typeLine(tm, "messenger")
	out := tm.(Model)
	if out.msgOpen {
		t.Fatalf("a deck without service must not open the panel")
	}
	if joined := strings.Join(out.shellEntries, "\n"); !strings.Contains(joined, "no service") {
		t.Fatalf("the refusal should land in the scrollback: %q", joined)
	}
}

// The game boots into the terminal (main calls BootIntoDeck): marduk's
// brief is announced before the player types anything, and the first
// Esc lands in the lair with the intro waiting in the LOG.
func TestBootIntoDeck(t *testing.T) {
	w := game.NewWorld()
	lipgloss.SetColorProfile(termenv.ANSI)
	var mod tea.Model = New(engine.New(w), "intro text")
	mm := mod.(Model)
	mm.BootIntoDeck() // main.go does this before the program runs
	mod = mm
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	mm = mod.(Model)
	if mm.shell == nil {
		t.Fatalf("boot should open the terminal")
	}
	v := mod.View()
	if !strings.Contains(v, "CYBERDECK // DECK") {
		t.Fatalf("boot should land on the terminal screen: %q", v)
	}
	if joined := strings.Join(mm.shellEntries, "\n"); !strings.Contains(joined, "[messenger] 1 unread") {
		t.Fatalf("the brief should be announced at boot: %q", joined)
	}

	mod, _ = mod.Update(spec(tea.KeyEsc))
	mm = mod.(Model)
	if mm.shell != nil {
		t.Fatalf("esc should log out into the lair")
	}
	if after := mod.View(); !strings.Contains(after, "intro text") {
		t.Fatalf("the intro should be waiting in the LOG after logout: %q", after)
	}
}
