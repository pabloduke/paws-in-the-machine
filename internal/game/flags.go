package game

// Story flags, declared once per docs/BOUNDARIES.md: every flag has
// exactly one writer-owner, and raw flag strings never appear outside
// this file. These are content-owned flags (set by hooks and dialogue
// effects declared in this package); a flag owned by a system would
// be exported from that system's package instead.
const (
	// Set by hacking hooks in the starter net (net.go).
	flagHeardWhisper    = "heard_whisper"
	flagGotSunFragment  = "got_sun_fragment"
	flagRanDig          = "ran_dig"
	flagWhisperSilenced = "whisper_silenced"

	// Set by verb hooks and dialogue effects.
	flagMugDown         = "mug_down"
	flagDrawerOpen      = "drawer_open"
	flagBaristaSoftened = "barista_softened"
	flagBaristaSawShard = "barista_saw_shard"

	// Once-markers for event rules (docs/systems/events.md).
	flagSeenWhisperReaction = "seen_whisper_reaction"
	flagSeenCounterEmpty    = "seen_counter_empty"
)

// One-shot XP award markers (awardOnce / dialogue.AwardOnce).
const (
	flagXPMug          = "xp_mug"
	flagXPShard        = "xp_shard"
	flagXPBaristaCharm = "xp_barista_charm"
	flagXPBaristaShard = "xp_barista_shard"
	flagXPLaptop       = "xp_laptop"
	flagXPHum          = "xp_hum"
)
