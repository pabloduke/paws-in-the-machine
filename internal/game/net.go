package game

import "github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"

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
						hacking.File("notes.txt",
							"(Placeholder) paw-scrawled notes:\n"+
								"the whisper came in off the old net. the relay\n"+
								"never really died. start there:  curl undernet.relay"),
					),
				),
				hacking.Dir("bin"),
			),
		},
		"undernet.relay": {
			Name:   "undernet.relay",
			Home:   "/",
			Banner: "(Placeholder) UNDERNET RELAY — abandoned, but listening.",
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
					"  okuda.grid    [410 gone]",
			},
		},
		"sunfarm.arc": {
			Name:   "sunfarm.arc",
			Home:   "/",
			Banner: "(Placeholder) sunfarm archive node. dust on everything.",
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
