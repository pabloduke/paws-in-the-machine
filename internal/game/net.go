package game

import (
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// starterNet is the net behind Buddy's deck: the hosts, files, and
// processes of the first hacking beat (docs/systems/hacking.md).
// Declared here like rooms are; the hooks set flags only.
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

func netJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagKnowsMicroslopPassword, Text: "The barista said Microslop contractor boxes were reset to `apple`."},
		{Flag: flagMicroslopRouteOpen, Text: "The coffee shop rack puts the deck somewhere Microslop can hear it."},
		{Flag: flagReadSunNotice, Text: "Microslop buried old sun liability under grid asset `SUNFARM-ARC`."},
		{Flag: flagGotSunNotice, Text: "Copied Microslop's sunlight liability notice onto the deck."},
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
