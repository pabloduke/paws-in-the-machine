package game

import (
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// starterNet is the net behind Buddy's deck: the hosts, files, and
// processes of the first hacking beat (docs/systems/hacking.md).
// Declared here like rooms are; the hooks set flags only.
// defaultAliases seeds the deck's ~/.aliases: friendlier verbs so a
// player who has never touched a shell can still play (list, look,
// search), while the real commands keep working. A pro reads the file
// (ls -a) and edits it — the comments show how. Plain `alias` syntax,
// the way anyone off Linux would expect.
const defaultAliases = `# Your shell aliases: friendlier names for the built-in commands.
# The real commands (ls, cat, grep...) always work too.
# Add your own, plain shell syntax:  alias name=command
#   e.g.  alias deep='grep -ir'
alias list=ls
alias look=cat
alias read=cat
alias search=grep
alias copy=cp
alias connect=ssh
alias nmap=scan
alias talk=messenger
alias scp=send
`

func starterNet() map[string]*hacking.Host {
	return map[string]*hacking.Host{
		"deck": {
			Name:     "deck",
			Username: "paws_in_the_machine",
			Home:     "/home/paws_in_the_machine",
			Root: hacking.Dir("/",
				hacking.Dir("home",
					hacking.Dir("paws_in_the_machine",
						hacking.Dir("notes",
							hacking.DynamicFile("notes.md", buddyNotes),
							// Mission 0: the bootstrap note, present from the
							// first boot and gone the moment its job is done —
							// it teaches opening the messenger, which reads
							// marduk's brief and downloads mission 1.
							&hacking.Node{
								Name:       "use_the_messenger.md",
								AbsentWhen: flagMissionMicroslop,
								Text:       missionUseMessenger,
							},
							// Mission files land here on their own when the
							// handler's briefing is read (PresentWhen +
							// Msg.Grants). Mission 1: marduk's Microslop job.
							&hacking.Node{
								Name:        "Microslop_Find_The_Layoff_List.md",
								PresentWhen: flagMissionMicroslop,
								TextFn:      missionMicroslopDoc,
							},
						),
						hacking.File(".aliases", defaultAliases),
					),
				),
				hacking.Dir("bin"),
			),
		},
		"undernet.relay": {
			Name:   "undernet.relay",
			Home:   "/",
			Banner: "(Placeholder) UNDERNET RELAY — abandoned, but listening.",
			Services: []*hacking.Service{
				{Port: 22, Protocol: hacking.ProtocolSSH, State: hacking.StateOpen},
				{Port: 80, Protocol: hacking.ProtocolHTTP, State: hacking.StateOpen},
			},
			Root: hacking.Dir("/",
				hacking.Dir("var",
					hacking.Dir("log",
						hacking.File("relay.log",
							"(Placeholder) 03:12 handshake from nobody\n"+
								"03:44 sunfarm.arc keeps answering. nobody asks anymore.\n"+
								"04:01 rain on the uplink again"),
					),
				),
			),
			Served: map[string]string{
				"/": "(Placeholder) mirror index — dead links mostly\n" +
					"  sunfarm.arc   [still answers]\n" +
					"  microslop     [corp intranet mirror]\n" +
					"  okuda.grid    [410 gone]",
			},
		},
		"microslop": {
			Name:     "microslop",
			Username: "jane_doe",
			Home:     "/",
			Password: "apple",
			Banner:   "(Placeholder) MICROSLOP CORP intranet. everything asks permission except the dust.",
			Services: []*hacking.Service{
				{Port: 21, Protocol: hacking.ProtocolFTP, State: hacking.StateOpen},
				{Port: 22, Protocol: hacking.ProtocolSSH, State: hacking.StateOpen, Password: "apple"},
				{Port: 23, Protocol: hacking.ProtocolTelnet, State: hacking.StateClosed},
			},
			Root: hacking.Dir("/",
				hacking.Dir("home",
					hacking.File("readme.txt",
						"(Placeholder) corp mirror residue:\n"+
							"search the logs for sun. nobody deletes anything here, they just rename it."),
				),
				hacking.Dir("var",
					hacking.Dir("log",
						hacking.File("access.log",
							"(Placeholder) 01:02 cafeteria bot accepted badge\n"+
								"02:40 HR matched employee 1008476 to /srv/hr/rif_q3.txt\n"+
								"03:17 sun notice moved to /srv/archive/sun_notice.txt\n"+
								"03:18 legal requested wording review"),
					),
				),
				hacking.Dir("srv",
					// Mission 1's target (docs/draft.md): the layoff
					// plans. RIF is corp-speak for "reduction in force" —
					// the barista's team is on the list.
					hacking.Dir("hr",
						&hacking.Node{
							Name:   "rif_q3.txt",
							OnRead: flagReadLayoffPlans,
							OnCopy: flagGotLayoffPlans,
							OnSend: flagLayoffPlansDelivered,
							Text: "(Placeholder) MICROSLOP HR — CONFIDENTIAL\n" +
								"Subject: Q3 reduction in force, wave two\n" +
								"Per legal: use \"role realignment\" in all comms.\n" +
								"Wave one (complete): contractor support, cafe " +
								"annex staff.\n" +
								"Wave two (pending): grid maintenance, archive " +
								"ops, night engineering.\n" +
								"Do not notify affected teams before the " +
								"badge-revoke batch runs.",
						},
					),
					hacking.Dir("archive",
						&hacking.Node{
							Name:   "sun_notice.txt",
							OnRead: flagReadSunNotice,
							OnCopy: flagGotSunNotice,
							Text: "(Placeholder) MICROSLOP INTERNAL NOTICE\n" +
								"Subject: sunlight liability exposure\n" +
								"The old sun project remains buried under grid asset SUNFARM-ARC.",
						},
					),
				),
			),
		},
		// The terminal half of issue #16: port 22 scans closed until
		// the records-office console in the overworld throws the port
		// switch (okuda.go) — the closed-ports loop.
		"okuda.grid": {
			Name:   "okuda.grid",
			Home:   "/",
			Banner: "(Placeholder) OKUDA GRID NODE — asset 410. this node was removed from inventory. it did not notice.",
			Services: []*hacking.Service{
				{Port: 22, Protocol: hacking.ProtocolSSH, State: hacking.StateOpen,
					OpenWhen: flagOkudaPortOpen},
			},
			Root: hacking.Dir("/",
				hacking.Dir("var",
					hacking.Dir("log",
						hacking.File("decommission.log",
							"(Placeholder) 09:00 asset 410 scheduled for erasure\n"+
								"09:41 records office cleared. physical port disabled.\n"+
								"09:42 archive door maglock engaged. badge table dropped.\n"+
								"09:43 erasure marked complete. nobody unplugged anything."),
					),
				),
				hacking.Dir("srv",
					hacking.Dir("vault",
						&hacking.Node{
							Name:   "410.txt",
							OnRead: flagReadOkuda410,
							Text: "(Placeholder) OKUDA INTERNAL — DO NOT MIGRATE\n" +
								"Asset 410 holds the records Okuda was paid to lose.\n" +
								"(What those records say is a story decision — content TBD.)",
						},
					),
					// The first door opened from inside the net: the
					// archive maglock answers this node and nothing else
					// (the badge table is gone). Gated exit in okuda.go.
					hacking.Dir("ctl",
						&hacking.Node{
							Name:  "unlock.bin",
							OnRun: flagOkudaAnnexUnlocked,
							RunText: "(Placeholder) maglock 02 [ARCHIVE] release... ok\n" +
								"badge table missing. lock will not re-arm.\n" +
								"somewhere above you, a door stops holding its breath.",
						},
					),
				),
			),
		},
		"sunfarm.arc": {
			Name:   "sunfarm.arc",
			Home:   "/",
			Banner: "(Placeholder) sunfarm archive node. dust on everything.",
			Services: []*hacking.Service{
				{Port: 22, Protocol: hacking.ProtocolSSH, State: hacking.StateOpen},
			},
			Root: hacking.Dir("/",
				hacking.Dir("var",
					hacking.Dir("log",
						&hacking.Node{
							Name:   "burial.log",
							OnRead: flagHeardWhisper,
							Text: "(Placeholder) ...they buried the sun...\n" +
								"...they buried the sun under the grid and told the rain to forget...",
						},
					),
				),
				hacking.Dir("srv",
					hacking.Dir("archive",
						&hacking.Node{
							Name:   "sun.frag",
							OnCopy: flagGotSunFragment,
							Text:   "(Placeholder) SUN//frag 1 of ? :: coordinates smeared by time",
						},
						&hacking.Node{
							Name:    "dig.bin",
							OnRun:   flagRanDig,
							RunText: "(Placeholder) unearthing... 3%... 61%... interrupted.\nthe fragment is enough for tonight.",
						},
					),
				),
			),
			Procs: []*hacking.Process{
				{PID: 27, Name: "whisperd", OnKill: flagWhisperSilenced},
			},
		},
	}
}

// netEvents pays XP for terminal beats (issue #24): hacks are worth
// what content says they're worth (docs/systems/hacking.md — no rolls,
// no formula). The rules poll at the checkpoint, and rules don't run
// while the terminal is open, so the payout lands when Buddy logs out
// — the session's haul, tallied at the door. Amounts echo the
// overworld's scale and are tuning data for #27.
func netEvents() []engine.When {
	beat := func(flag, once string, xp int, text string) engine.When {
		return engine.When{
			Flags: []string{flag},
			Once:  once,
			Do: func(w *engine.World) string {
				return text + "\n\n" + engine.AwardXP(w, xp)
			},
		}
	}
	return []engine.When{
		beat(flagHeardWhisper, flagXPWhisper, 3,
			"(Placeholder) The whisper in the dead code follows you off "+
				"the deck. Worth knowing. Worth more, later."),
		beat(flagGotSunFragment, flagXPSunFragment, 5,
			"(Placeholder) A fragment of the sun sits on your deck now, "+
				"warm in the way data shouldn't be."),
		beat(flagRanDig, flagXPRanDig, 2,
			"(Placeholder) dig.bin got further than anything has in "+
				"years before the archive cut it off."),
		beat(flagWhisperSilenced, flagXPWhisperQuiet, 2,
			"(Placeholder) whisperd is quiet. The archive feels less "+
				"haunted, and somehow that's worse."),
		beat(flagReadSunNotice, flagXPSunNoticeRead, 3,
			"(Placeholder) Microslop's lawyers knew about the sun. It's "+
				"in writing. You read the writing."),
		beat(flagGotSunNotice, flagXPSunNoticeCopied, 5,
			"(Placeholder) The liability notice is on your deck now — "+
				"corp ink, cat claws."),
		beat(flagReadOkuda410, flagXPOkuda410, 5,
			"(Placeholder) Asset 410, read at the source. Whatever Okuda "+
				"was paid to lose, you found the receipt."),
		beat(flagReadLayoffPlans, flagXPLayoffRead, 3,
			"(Placeholder) \"Role realignment.\" You read the real words "+
				"under the nice ones."),
		beat(flagGotLayoffPlans, flagXPLayoffCopied, 5,
			"(Placeholder) The layoff plans are on your deck. Names, "+
				"dates, the badge-revoke batch. Heavier than data should be."),
		beat(flagLayoffPlansDelivered, flagXPLayoffDelivered, 8,
			"(Placeholder) The drop took the plans. Somewhere, marduk is "+
				"already reading. Mission one, done."),
	}
}

func netJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagReadMicroslopBadge, Text: "The barista's old Microslop badge has username `jane_doe`, employee ID `1008476`, and password `apple` tucked behind it."},
		{Flag: flagReadLayoffPlans, Text: "Found Microslop's Q3 layoff plans in /srv/hr/rif_q3.txt — \"role realignment,\" wave two pending."},
		{Flag: flagGotLayoffPlans, Text: "Copied the layoff plans onto the deck."},
		{Flag: flagLayoffPlansDelivered, Text: "Sent the layoff plans to marduk's drop. Mission one complete."},
		{Flag: flagReadSunNotice, Text: "Microslop buried old sun liability under grid asset `SUNFARM-ARC`."},
		{Flag: flagGotSunNotice, Text: "Copied Microslop's sunlight liability notice onto the deck."},
		{Flag: flagReadOkuda410, Text: "(Placeholder) okuda.grid is asset 410: records Okuda was paid to lose, still humming in their own records office."},
	}
}

// missionUseMessenger is mission 0: the very first note, teaching the
// one thing the player must do to get everything else — open the
// messenger. It vanishes the moment they do (net.go AbsentWhen).
const missionUseMessenger = `# Getting Started

Someone's trying to reach you. Your messenger is blinking.

## Steps

- [ ] open the messenger: type ` + "`messenger`" + ` (or ` + "`talk`" + `)
      and read what's waiting`

// missionMicroslopSteps is mission 1's checklist, rendered into the
// mission file (not notes.md). Mission 1 is the explicit end of the
// hint-fade (docs/draft.md): these read like orders; missions 2 and 3
// get vaguer, and from 4 on notes are just intel.
var missionMicroslopSteps = []struct {
	flag string
	text string
}{
	{flagReadMicroslopBadge, "meow at the barista, then look at the post-it behind her old badge; or sneak to the post-it"},
	{flagGotLayoffPlans, "return to the lair, connect with ssh jane_doe@microslop, search for employee 1008476, and copy the layoff plans home"},
	{flagLayoffPlansDelivered, "send the plans to marduk: send <file>"},
}

// missionMicroslopDoc renders the mission file: params up top, a
// checklist that ticks as flags land. Present only once marduk's brief
// is read (net.go PresentWhen), so it "downloads" from the handler.
func missionMicroslopDoc(w *engine.World) string {
	var b strings.Builder
	b.WriteString("# Microslop — Find the Layoff List\n\n")
	b.WriteString("From: marduk\n")
	b.WriteString("Get inside Microslop's intranet, pull the real layoff " +
		"plans, and send them back.\n\n## Steps\n")
	done := 0
	for _, step := range missionMicroslopSteps {
		mark := "[ ]"
		if w.Flags[step.flag] {
			mark = "[x]"
			done++
		}
		b.WriteString("\n- " + mark + " " + step.text)
	}
	if done == len(missionMicroslopSteps) {
		b.WriteString("\n\nMission complete. marduk has the plans.")
	}
	return b.String()
}

// buddyNotes renders ~/notes/notes.md: Buddy's field notes, derived
// from discovered flags. Missions live in their own files now (user
// ruling 2026-07-10); this is just what he's learned.
func buddyNotes(w *engine.World) string {
	entries := w.JournalEntries()
	if len(entries) == 0 {
		return "# Notes\n\nNothing solid yet."
	}
	var b strings.Builder
	b.WriteString("# Notes\n")
	for _, e := range entries {
		b.WriteString("\n- " + e)
	}
	return b.String()
}
