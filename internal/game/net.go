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
`

func starterNet() map[string]*hacking.Host {
	return map[string]*hacking.Host{
		"deck": {
			Name: "deck",
			Home: "/home/paws_in_the_machine",
			Root: hacking.Dir("/",
				hacking.Dir("home",
					hacking.Dir("paws_in_the_machine",
						hacking.Dir("notes",
							hacking.DynamicFile("notes.md", buddyNotes),
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
			Home:     "/",
			Password: "apple",
			Require:  flagMicroslopRouteOpen,
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
								"03:17 sun notice moved to /srv/archive/sun_notice.txt\n"+
								"03:18 legal requested wording review"),
					),
				),
				hacking.Dir("srv",
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
		// The terminal half of issue #16: port 22 scans filtered until
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
	}
}

func netJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagKnowsMicroslopPassword, Text: "The barista said Microslop contractor boxes were reset to `apple`."},
		{Flag: flagMicroslopRouteOpen, Text: "Bridged the coffee shop rack onto the lair's uplink. The deck can reach Microslop now."},
		{Flag: flagReadSunNotice, Text: "Microslop buried old sun liability under grid asset `SUNFARM-ARC`."},
		{Flag: flagGotSunNotice, Text: "Copied Microslop's sunlight liability notice onto the deck."},
		{Flag: flagReadOkuda410, Text: "(Placeholder) okuda.grid is asset 410: records Okuda was paid to lose, still humming in their own records office."},
	}
}

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
