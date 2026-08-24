package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const NetworkAssignmentsVersion = 1

type NetworkAssignment struct {
	TerminalID    string `json:"terminal_id"`
	HostNetworkID string `json:"host_network_id"`
}

type NetworkAssignmentsFile struct {
	Version     int                 `json:"version"`
	Assignments []NetworkAssignment `json:"assignments"`
}

func EmptyNetworkAssignments() NetworkAssignmentsFile {
	return NetworkAssignmentsFile{Version: NetworkAssignmentsVersion, Assignments: []NetworkAssignment{}}
}

func DecodeNetworkAssignments(data []byte) (NetworkAssignmentsFile, error) {
	var file NetworkAssignmentsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return NetworkAssignmentsFile{}, fmt.Errorf("network assignments: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return NetworkAssignmentsFile{}, fmt.Errorf("network assignments: %w", err)
	}
	if file.Assignments == nil {
		return NetworkAssignmentsFile{}, fmt.Errorf("network assignments: assignments must be an array")
	}
	if err := ValidateNetworkAssignments(file); err != nil {
		return NetworkAssignmentsFile{}, err
	}
	return file, nil
}

func EncodeNetworkAssignments(file NetworkAssignmentsFile) ([]byte, error) {
	if file.Assignments == nil {
		file.Assignments = []NetworkAssignment{}
	}
	if err := ValidateNetworkAssignments(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("network assignments: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateNetworkAssignments(file NetworkAssignmentsFile) error {
	if file.Version != NetworkAssignmentsVersion {
		return fmt.Errorf("network assignments: file version %d, want %d", file.Version, NetworkAssignmentsVersion)
	}
	seenTerminals := make(map[string]struct{}, len(file.Assignments))
	for i, assignment := range file.Assignments {
		where := fmt.Sprintf("network assignments: assignment %d", i+1)
		if !validUUID(assignment.TerminalID) {
			return fmt.Errorf("%s has invalid terminal UUID %q", where, assignment.TerminalID)
		}
		if !validUUID(assignment.HostNetworkID) {
			return fmt.Errorf("%s has invalid host-network UUID %q", where, assignment.HostNetworkID)
		}
		if _, exists := seenTerminals[assignment.TerminalID]; exists {
			return fmt.Errorf("%s duplicates terminal UUID %q", where, assignment.TerminalID)
		}
		seenTerminals[assignment.TerminalID] = struct{}{}
	}
	return nil
}
