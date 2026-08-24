package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
)

const TerminalsVersion = 1

var (
	usernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	hostNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_.-]*[a-z0-9])?$`)
)

type Terminal struct {
	ID       string `json:"id"`
	Username string `json:"username,omitempty"` // Legacy v1 data; access now comes from terminal_access.json.
	HostName string `json:"host_name"`
}

type TerminalsFile struct {
	Version   int        `json:"version"`
	Terminals []Terminal `json:"terminals"`
}

func EmptyTerminals() TerminalsFile {
	return TerminalsFile{Version: TerminalsVersion, Terminals: []Terminal{}}
}

func DecodeTerminals(data []byte) (TerminalsFile, error) {
	var file TerminalsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return TerminalsFile{}, fmt.Errorf("terminals: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return TerminalsFile{}, fmt.Errorf("terminals: %w", err)
	}
	if file.Terminals == nil {
		return TerminalsFile{}, fmt.Errorf("terminals: terminals must be an array")
	}
	if err := ValidateTerminals(file); err != nil {
		return TerminalsFile{}, err
	}
	return file, nil
}

func EncodeTerminals(file TerminalsFile) ([]byte, error) {
	if file.Terminals == nil {
		file.Terminals = []Terminal{}
	}
	if err := ValidateTerminals(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("terminals: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateTerminals(file TerminalsFile) error {
	if file.Version != TerminalsVersion {
		return fmt.Errorf("terminals: file version %d, want %d", file.Version, TerminalsVersion)
	}
	seenIDs := make(map[string]struct{}, len(file.Terminals))
	for i, terminal := range file.Terminals {
		where := fmt.Sprintf("terminals: terminal %d", i+1)
		if !validUUID(terminal.ID) {
			return fmt.Errorf("%s has invalid UUID %q", where, terminal.ID)
		}
		if _, exists := seenIDs[terminal.ID]; exists {
			return fmt.Errorf("%s duplicates UUID %q", where, terminal.ID)
		}
		seenIDs[terminal.ID] = struct{}{}
		if terminal.Username != "" && !usernamePattern.MatchString(terminal.Username) {
			return fmt.Errorf("%s has invalid username %q", where, terminal.Username)
		}
		if !hostNamePattern.MatchString(terminal.HostName) {
			return fmt.Errorf("%s has invalid host name %q", where, terminal.HostName)
		}
	}
	return nil
}

func ValidTerminalUsername(username string) bool { return usernamePattern.MatchString(username) }

func ValidTerminalHostName(hostName string) bool { return hostNamePattern.MatchString(hostName) }
