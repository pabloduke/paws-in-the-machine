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
	flagReadSunNotice   = "read_microslop_sun_notice"
	flagGotSunNotice    = "got_microslop_notice"

	// Set by verb hooks and dialogue effects.
	flagMugDown                = "mug_down"
	flagDrawerOpen             = "drawer_open"
	flagBaristaSoftened        = "barista_softened"
	flagBaristaSawShard        = "barista_saw_shard"
	flagKnowsMicroslopPassword = "knows_microslop_password"
	flagMicroslopRouteOpen     = "microslop_route_open"

	// Okuda HQ (okuda.go). The port flag is the overworld half of the
	// first overworld↔terminal handshake: the office console sets it,
	// okuda.grid's SSH service reads it (net.go).
	flagOkudaPortOpen     = "okuda_port_open"
	flagSawLobbyFlyer     = "saw_lobby_flyer"
	flagPlayedLostCat     = "played_lost_cat"
	flagCorridorLightsOut = "corridor_lights_out"
	flagReadOkuda410      = "read_okuda_410"
	// Set by running unlock.bin on okuda.grid (net.go); read by the
	// records office's gated archive door (okuda.go) — the first door
	// opened from inside the net.
	flagOkudaAnnexUnlocked = "okuda_annex_unlocked"

	// Set by check failures and read back as modifiers/events
	// (docs/systems/stealth.md: failure is a story state).
	flagHoundAlerted = "hound_alerted"
	// Set by knocking the espresso machine; consumed by the sneak
	// roll it distracts for.
	flagMugShattered = "mug_shattered"
	// Set by asking the softened barista to call the hound over for
	// scraps: the hound stops watching the back door (perception,
	// docs/systems/stealth.md) until something ends snack time.
	flagHoundLured = "hound_lured"
	// Okuda check failures (okuda.go): a refused cat becomes a
	// situation; a spooked guard stiffens both routes in.
	flagReceptionistSuspicious = "receptionist_suspicious"
	flagOkudaGuardAlerted      = "okuda_guard_alerted"

	// Once-markers for event rules (docs/systems/events.md).
	flagSeenWhisperReaction        = "seen_whisper_reaction"
	flagSeenCounterEmpty           = "seen_counter_empty"
	flagSeenHoundAlerted           = "seen_hound_alerted"
	flagSeenReceptionistSuspicious = "seen_receptionist_suspicious"
	flagSeenOkudaGuardAlerted      = "seen_okuda_guard_alerted"
)

// One-shot XP award markers (awardOnce / dialogue.AwardOnce).
const (
	flagXPMug          = "xp_mug"
	flagXPShard        = "xp_shard"
	flagXPBaristaCharm = "xp_barista_charm"
	flagXPBaristaShard = "xp_barista_shard"
	flagXPLaptop       = "xp_laptop"
	flagXPHum          = "xp_hum"
	flagXPHoundLure    = "xp_hound_lure"
	flagXPLobbyFlyer   = "xp_lobby_flyer"
	flagXPOkudaConsole = "xp_okuda_console"
)
