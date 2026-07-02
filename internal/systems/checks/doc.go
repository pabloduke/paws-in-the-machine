// Package checks is the dice of Paws in the Machine — the only dice in
// the game. Full spec: docs/systems/stealth.md.
//
// A check resolves an approach against an obstacle:
//
//	hidden roll + stat  vs  difficulty
//
// Rolls are seeded (the XCOM rule): the outcome is a pure function of
// (world seed, check identity, stat), so an identical attempt gives an
// identical result and save-scumming is useless. The roll changes only
// when an input changes — a raised stat, a different approach, a
// different check. Dice and difficulties are never shown to the player;
// circumstances are telegraphed in prose by content.
//
// The exposed surface is Check (roll resolution) and Guarded (the
// component content attaches to an obstacle entity, declaring its
// approach matrix: which approaches are possible, at what difficulty,
// with what prose). Everything else is internal.
package checks
