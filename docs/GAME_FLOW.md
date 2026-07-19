# Game Flow

This is the canonical code-derived map of implemented game behavior. Update
it in the same change as any game-logic change. Solid arrows represent
required state transitions; dashed arrows represent optional actions or
knowledge-based shortcuts that the runtime permits.

```mermaid
flowchart TD
    subgraph Startup[Startup and mission bootstrap]
        Boot([Program starts]) --> Deck[BootIntoDeck opens the deck terminal]
        Deck --> Note[use_the_messenger.md is present]
        Deck --> Welcome[marduk welcome message is unread]
        Note -.-> ReadNote[cat use_the_messenger.md]
        ReadNote -.-> OpenMessenger[Run messenger or talk]
        Welcome --> OpenMessenger
        Deck -->|Bootstrap note may be skipped| OpenMessenger
        OpenMessenger --> MissionFlag[Set mission_microslop]
        MissionFlag --> NoteSwap[Bootstrap note disappears]
        MissionFlag --> MissionFile[Mission 1 file appears in notes]
    end

    subgraph Credential[Microslop credential]
        Deck --> Exit[exit or logout]
        Exit --> Lair[Buddy's Lair]
        Lair --> Coffee[Go north to the Coffee Shop]
        Coffee --> Route{Post-it approach}

        Route -->|Charm| Purr[Purr with Charm at least 8]
        Purr --> Softened[Set barista_softened]
        Softened --> NotBurned{barista_burned absent?}
        NotBurned -->|Yes| Examine[Look, read, or examine post-it]
        NotBurned -->|No| Sneak

        Route -->|Cruel dialogue choice| TipJar[Knock over tip jar]
        TipJar --> Burned[Set barista_burned]
        Burned --> Sneak[Sneak post-it at difficulty 22]

        Route -.->|Optional setup| Mug[Knock espresso machine mug]
        Mug -->|One check gets +4| Sneak
        Route -->|Stealth| Sneak
        Sneak -->|Success| Badge[Set read_microslop_badge]
        Sneak -->|Failure| Burned
        Examine --> Badge
        Badge --> Intel[Notes expose employee 1008476 and password apple]
    end

    subgraph Mission1[Microslop delivery]
        Intel --> Return[Return south to the lair]
        Return --> UseDeck[use deck]
        UseDeck -.-> Scan[scan microslop]
        Scan -.-> SSH[ssh microslop]
        UseDeck --> SSH
        Deck -.->|Player already knows apple| SSH
        SSH --> Password[Enter apple]
        Password --> Connected[Connected to microslop]
        Connected -.-> Search[grep -ir 1008476 /]
        Search --> Target["/srv/hr/rif_q3.txt"]
        Target -.-> ReadPlans[cat file sets read_layoff_plans]
        ReadPlans -.-> CopyPlans[Copy file to deck]
        Target -->|Reading is not required| CopyPlans
        CopyPlans --> GotPlans[Set got_layoff_plans]
        GotPlans --> Send[send deck copy]
        Send --> Delivered[Set layoff_plans_delivered]
        Delivered --> Reply[marduk reply arrives]
        Reply --> Complete[Mission file renders Mission complete]
    end

    subgraph Independent[Independent executable branches]
        Exit --> Hubs[All hubs are travelable]
        Hubs --> Plaza[Plaza and arcade exploration]
        Hubs --> Okuda[Okuda HQ]

        Okuda -->|Lobby Charm| Office[Records Office]
        Okuda -->|Fire escape and corridor checks| Office
        Office --> Port[Use console; set okuda_port_open]
        Port --> OkudaHost[Return to deck; ssh okuda.grid]
        OkudaHost --> Read410[Read vault 410 file]
        OkudaHost --> Unlock[Run unlock.bin]
        Unlock --> AnnexFlag[Set okuda_annex_unlocked]
        AnnexFlag --> Annex[Enter Deep Archive]
        Annex --> ArchiveStub[(Placeholder: archive content)]

        Deck --> Sunfarm[ssh sunfarm.arc]
        Sunfarm --> Whisper[Read burial.log]
        Sunfarm --> Fragment[Copy sun.frag]
        Sunfarm --> Dig[Run dig.bin]
        Sunfarm --> Silence[Kill whisperd]
        Whisper --> SunEvents[Derived messages, notes, and logout XP]
        Fragment --> SunEvents
        Dig --> SunEvents
        Silence --> SunEvents

        Connected -.-> Notice[Read or copy sun_notice.txt]
        Notice --> NoticeEvents[Derived message, notes, and logout XP]
    end
```

## Runtime properties exposed by the diagram

- The bootstrap note and `scan microslop` are guidance, not gates.
- `read_microslop_badge` does not gate SSH; knowing `apple` bypasses the
  coffee-shop route.
- `mission_microslop` does not gate the target file or its hooks, so the
  delivery flags can land before the briefing is read.
- Okuda, sunfarm, and Plaza behavior is independently reachable rather than
  part of Mission 1.
- The Deep Archive is reachable, but its terminal content remains a marked
  placeholder in code.
