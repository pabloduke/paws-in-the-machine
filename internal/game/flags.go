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
	// Mission 1 (docs/draft.md): the layoff plans on microslop's HR
	// share — read at the source, copied home, sent to marduk's drop.
	flagReadLayoffPlans      = "read_layoff_plans"
	flagGotLayoffPlans       = "got_layoff_plans"
	flagLayoffPlansDelivered = "layoff_plans_delivered"
	// Set when marduk's brief is read (Msg.Grants): the mission file
	// appears in ~/notes (PresentWhen). "Downloading" the mission.
	flagMissionMicroslop = "mission_microslop"

	// Set by verb hooks and dialogue effects.
	flagMugDown         = "mug_down"
	flagBaristaSoftened = "barista_softened"
	// NPC memory (#14): a per-NPC fact the barista carries about how
	// Buddy treated her at first contact. Warm memory is the existing
	// flagBaristaSoftened; this is its cold twin. Read by her greeting
	// (cold line), her description, and the invited post-it look. Kept
	// off the critical path on purpose:
	// burning her closes the invited post-it look, but the stealth route
	// remains open (failure is a fork, not a wall).
	flagBaristaBurned      = "barista_burned"
	flagReadMicroslopBadge = "read_microslop_badge"

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
	// Set by knocking the espresso machine; consumed by the sneak
	// roll it distracts for.
	flagMugShattered = "mug_shattered"
	// Okuda check failures (okuda.go): a refused cat becomes a
	// situation; a spooked guard stiffens both routes in.
	flagReceptionistSuspicious = "receptionist_suspicious"
	flagOkudaGuardAlerted      = "okuda_guard_alerted"

	// Once-markers for event rules (docs/systems/events.md).
	flagSeenWhisperReaction        = "seen_whisper_reaction"
	flagSeenReceptionistSuspicious = "seen_receptionist_suspicious"
	flagSeenOkudaGuardAlerted      = "seen_okuda_guard_alerted"
)

// One-shot XP award markers (awardOnce / dialogue.AwardOnce).
const (
	flagXPMug          = "xp_mug"
	flagXPShard        = "xp_shard"
	flagXPBaristaCharm = "xp_barista_charm"
	flagXPLaptop       = "xp_laptop"
	flagXPHum          = "xp_hum"
	flagXPLobbyFlyer   = "xp_lobby_flyer"
	flagXPOkudaConsole = "xp_okuda_console"
	// Hacking payouts (netEvents, issue #24): once-markers for the
	// terminal beats that pay content-valued XP at logout.
	flagXPWhisper         = "xp_whisper"
	flagXPSunFragment     = "xp_sun_fragment"
	flagXPRanDig          = "xp_ran_dig"
	flagXPWhisperQuiet    = "xp_whisper_quiet"
	flagXPSunNoticeRead   = "xp_sun_notice_read"
	flagXPSunNoticeCopied = "xp_sun_notice_copied"
	flagXPOkuda410        = "xp_okuda_410"
	flagXPLayoffRead      = "xp_layoff_read"
	flagXPLayoffCopied    = "xp_layoff_copied"
	flagXPLayoffDelivered = "xp_layoff_delivered"
)
