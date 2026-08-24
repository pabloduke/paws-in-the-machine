package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const NPCsVersion = 1

type NPC struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type NPCsFile struct {
	Version int   `json:"version"`
	NPCs    []NPC `json:"npcs"`
}

func EmptyNPCs() NPCsFile { return NPCsFile{Version: NPCsVersion, NPCs: []NPC{}} }

func DecodeNPCs(data []byte) (NPCsFile, error) {
	var file NPCsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return NPCsFile{}, fmt.Errorf("npcs: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return NPCsFile{}, fmt.Errorf("npcs: %w", err)
	}
	if file.NPCs == nil {
		return NPCsFile{}, fmt.Errorf("npcs: npcs must be an array")
	}
	if err := ValidateNPCs(file); err != nil {
		return NPCsFile{}, err
	}
	return file, nil
}

func EncodeNPCs(file NPCsFile) ([]byte, error) {
	if file.NPCs == nil {
		file.NPCs = []NPC{}
	}
	if err := ValidateNPCs(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("npcs: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateNPCs(file NPCsFile) error {
	if file.Version != NPCsVersion {
		return fmt.Errorf("npcs: file version %d, want %d", file.Version, NPCsVersion)
	}
	seen := make(map[string]struct{}, len(file.NPCs))
	for i, npc := range file.NPCs {
		where := fmt.Sprintf("npcs: npc %d", i+1)
		if err := validateDescribedEntity(where, npc.ID, npc.Name, npc.Description, seen); err != nil {
			return err
		}
	}
	return nil
}
