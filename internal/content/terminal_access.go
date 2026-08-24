package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const TerminalAccessVersion = 1

type TerminalAccess struct {
	UserID     string `json:"user_id"`
	TerminalID string `json:"terminal_id"`
}

type TerminalAccessFile struct {
	Version int              `json:"version"`
	Access  []TerminalAccess `json:"access"`
}

func EmptyTerminalAccess() TerminalAccessFile {
	return TerminalAccessFile{Version: TerminalAccessVersion, Access: []TerminalAccess{}}
}

func DecodeTerminalAccess(data []byte) (TerminalAccessFile, error) {
	var file TerminalAccessFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return TerminalAccessFile{}, fmt.Errorf("terminal access: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return TerminalAccessFile{}, fmt.Errorf("terminal access: %w", err)
	}
	if file.Access == nil {
		return TerminalAccessFile{}, fmt.Errorf("terminal access: access must be an array")
	}
	if err := ValidateTerminalAccess(file); err != nil {
		return TerminalAccessFile{}, err
	}
	return file, nil
}

func EncodeTerminalAccess(file TerminalAccessFile) ([]byte, error) {
	if file.Access == nil {
		file.Access = []TerminalAccess{}
	}
	if err := ValidateTerminalAccess(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("terminal access: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateTerminalAccess(file TerminalAccessFile) error {
	if file.Version != TerminalAccessVersion {
		return fmt.Errorf("terminal access: file version %d, want %d", file.Version, TerminalAccessVersion)
	}
	seen := make(map[string]struct{}, len(file.Access))
	for i, access := range file.Access {
		where := fmt.Sprintf("terminal access: grant %d", i+1)
		if !validUUID(access.UserID) {
			return fmt.Errorf("%s has invalid user UUID %q", where, access.UserID)
		}
		if !validUUID(access.TerminalID) {
			return fmt.Errorf("%s has invalid terminal UUID %q", where, access.TerminalID)
		}
		key := access.UserID + "\x00" + access.TerminalID
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%s duplicates user/terminal pair", where)
		}
		seen[key] = struct{}{}
	}
	return nil
}
