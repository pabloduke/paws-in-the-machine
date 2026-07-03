// Package dialogue implements Fallout-style conversations: talking to
// an NPC enters a node graph where the player picks numbered choices.
//
// The system is deliberately separate from the parser. The parser only
// recognizes "talk <npc>"; dialogue owns choice visibility, conditions,
// effects, node transitions, and rendering.
package dialogue
