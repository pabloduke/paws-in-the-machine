package engine

import "strings"

// Command is a parsed player intent: a canonical verb plus an optional
// object phrase ("take shard" -> {Verb: "take", Object: "shard"}).
type Command struct {
	Verb   string
	Object string
}

// verbAliases maps every accepted input verb to its canonical form.
var verbAliases = map[string]string{
	"look": "look", "l": "look",
	"examine": "examine", "x": "examine", "inspect": "examine", "read": "examine",
	"go": "go", "walk": "go", "move": "go",
	"take": "take", "get": "take", "grab": "take", "pick": "take",
	"drop":      "drop",
	"inventory": "inventory", "i": "inventory", "inv": "inventory",
	"use": "use", "activate": "use",
	"knock": "knock", "push": "knock", "bat": "knock", "swat": "knock", "paw": "knock",
	"turn": "turn", "rotate": "turn", "twist": "turn",
	"sneak":   "sneak",
	"parkour": "parkour", "leap": "parkour", "vault": "parkour",
	"charm": "charm", "purr": "charm",
	"talk": "talk", "speak": "talk",
	"train":   "train",
	"stats":   "stats",
	"journal": "journal", "j": "journal",
	"help": "help", "?": "help",
	"quit": "quit", "q": "quit", "exit": "quit",
}

// directions are bare words treated as "go <direction>".
var directions = map[string]string{
	"north": "north", "n": "north",
	"south": "south", "s": "south",
	"east": "east", "e": "east",
	"west": "west", "w": "west",
	"up": "up", "u": "up",
	"down": "down", "d": "down",
	"in": "in", "enter": "in",
	"out": "out",
}

// noise words are dropped from the object phrase ("pick up the shard").
var noise = map[string]bool{
	"the": true, "a": true, "an": true, "at": true, "to": true,
	"up": true, "on": true, "in": true, "with": true, "into": true,
	"off": true, "past": true, "around": true, "by": true,
}

// Parse turns raw player input into a Command. It returns ok=false when
// the input is empty or the verb is unknown.
func Parse(input string) (Command, bool) {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(input)))
	if len(words) == 0 {
		return Command{}, false
	}

	// A bare direction is shorthand for movement.
	if dir, isDir := directions[words[0]]; isDir && len(words) == 1 {
		return Command{Verb: "go", Object: dir}, true
	}

	verb, known := verbAliases[words[0]]
	if !known {
		return Command{Verb: words[0]}, false
	}

	var object []string
	for _, word := range words[1:] {
		// For movement, direction words win over noise stripping
		// ("go up", "go in").
		if verb == "go" {
			if dir, isDir := directions[word]; isDir {
				object = append(object, dir)
				continue
			}
		}
		if noise[word] {
			continue
		}
		object = append(object, word)
	}
	return Command{Verb: verb, Object: strings.Join(object, " ")}, true
}
