package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const HostNetworksVersion = 1

type HostNetwork struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HostNetworksFile struct {
	Version      int           `json:"version"`
	HostNetworks []HostNetwork `json:"host_networks"`
}

func EmptyHostNetworks() HostNetworksFile {
	return HostNetworksFile{Version: HostNetworksVersion, HostNetworks: []HostNetwork{}}
}

func DecodeHostNetworks(data []byte) (HostNetworksFile, error) {
	var file HostNetworksFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return HostNetworksFile{}, fmt.Errorf("host networks: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return HostNetworksFile{}, fmt.Errorf("host networks: %w", err)
	}
	if file.HostNetworks == nil {
		return HostNetworksFile{}, fmt.Errorf("host networks: host_networks must be an array")
	}
	if err := ValidateHostNetworks(file); err != nil {
		return HostNetworksFile{}, err
	}
	return file, nil
}

func EncodeHostNetworks(file HostNetworksFile) ([]byte, error) {
	if file.HostNetworks == nil {
		file.HostNetworks = []HostNetwork{}
	}
	if err := ValidateHostNetworks(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("host networks: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateHostNetworks(file HostNetworksFile) error {
	if file.Version != HostNetworksVersion {
		return fmt.Errorf("host networks: file version %d, want %d", file.Version, HostNetworksVersion)
	}
	seen := make(map[string]struct{}, len(file.HostNetworks))
	for i, network := range file.HostNetworks {
		where := fmt.Sprintf("host networks: network %d", i+1)
		if !validUUID(network.ID) {
			return fmt.Errorf("%s has invalid UUID %q", where, network.ID)
		}
		if _, exists := seen[network.ID]; exists {
			return fmt.Errorf("%s duplicates UUID %q", where, network.ID)
		}
		seen[network.ID] = struct{}{}
		if strings.TrimSpace(network.Name) == "" {
			return fmt.Errorf("%s has an empty name", where)
		}
	}
	return nil
}
