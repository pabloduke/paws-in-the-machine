// Package hubs organizes the city into districts and moves Buddy
// between them. Full spec: docs/systems/hubs.md.
//
// A hub is an ordinary entity that is a direct child of the world root
// and carries the Hub component (its only config: the entry room ID).
// Rooms are the hub's children. Every hub is known and travelable from
// the start — Buddy has lived in this city his whole life; game state
// gates what is worth doing in a hub, never access to it.
//
// The exposed surface is the Hub component plus List, Current, and
// Travel. The UI renders List in a side panel and calls Travel when
// the player selects a destination.
package hubs
