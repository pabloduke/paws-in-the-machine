// Package dialogue implements Fallout/Mass Effect-style conversations:
// talking to an NPC enters a node graph where the player picks numbered
// choices. Stat-gated choices render visible with a derived "[Charm 8]"
// tag and stay locked (ME1-style) until the stat qualifies; flag/item
// gates hide choices entirely.
//
// The system is deliberately separate from the parser. The parser only
// recognizes "meow <npc>"; dialogue owns choice visibility, conditions,
// effects, node transitions, and rendering.
package dialogue
