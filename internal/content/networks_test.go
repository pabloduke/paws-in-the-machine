package content

import "testing"

func TestHostNetworksAndAssignmentsRoundTrip(t *testing.T) {
	networkID := "123e4567-e89b-42d3-a456-426614174010"
	terminalID := "123e4567-e89b-42d3-a456-426614174011"
	networks := HostNetworksFile{Version: HostNetworksVersion, HostNetworks: []HostNetwork{{ID: networkID, Name: "Network"}}}
	data, err := EncodeHostNetworks(networks)
	if err != nil {
		t.Fatal(err)
	}
	decodedNetworks, err := DecodeHostNetworks(data)
	if err != nil || len(decodedNetworks.HostNetworks) != 1 || decodedNetworks.HostNetworks[0] != networks.HostNetworks[0] {
		t.Fatalf("host-network round trip = %+v, %v", decodedNetworks, err)
	}

	assignments := NetworkAssignmentsFile{Version: NetworkAssignmentsVersion, Assignments: []NetworkAssignment{{TerminalID: terminalID, HostNetworkID: networkID}}}
	data, err = EncodeNetworkAssignments(assignments)
	if err != nil {
		t.Fatal(err)
	}
	decodedAssignments, err := DecodeNetworkAssignments(data)
	if err != nil || len(decodedAssignments.Assignments) != 1 || decodedAssignments.Assignments[0] != assignments.Assignments[0] {
		t.Fatalf("assignment round trip = %+v, %v", decodedAssignments, err)
	}
}

func TestHostNetworkAndAssignmentValidation(t *testing.T) {
	id := "123e4567-e89b-42d3-a456-426614174010"
	other := "123e4567-e89b-42d3-a456-426614174011"
	invalid := []error{
		ValidateHostNetworks(HostNetworksFile{Version: 2, HostNetworks: []HostNetwork{}}),
		ValidateHostNetworks(HostNetworksFile{Version: HostNetworksVersion, HostNetworks: []HostNetwork{{ID: "bad", Name: "Network"}}}),
		ValidateHostNetworks(HostNetworksFile{Version: HostNetworksVersion, HostNetworks: []HostNetwork{{ID: id, Name: " "}}}),
		ValidateHostNetworks(HostNetworksFile{Version: HostNetworksVersion, HostNetworks: []HostNetwork{{ID: id, Name: "One"}, {ID: id, Name: "Two"}}}),
		ValidateNetworkAssignments(NetworkAssignmentsFile{Version: 2, Assignments: []NetworkAssignment{}}),
		ValidateNetworkAssignments(NetworkAssignmentsFile{Version: NetworkAssignmentsVersion, Assignments: []NetworkAssignment{{TerminalID: "bad", HostNetworkID: id}}}),
		ValidateNetworkAssignments(NetworkAssignmentsFile{Version: NetworkAssignmentsVersion, Assignments: []NetworkAssignment{{TerminalID: other, HostNetworkID: "bad"}}}),
		ValidateNetworkAssignments(NetworkAssignmentsFile{Version: NetworkAssignmentsVersion, Assignments: []NetworkAssignment{{TerminalID: other, HostNetworkID: id}, {TerminalID: other, HostNetworkID: id}}}),
	}
	for i, err := range invalid {
		if err == nil {
			t.Errorf("invalid case %d was accepted", i)
		}
	}
}

func TestHostNetworkAndAssignmentDecodersAreStrict(t *testing.T) {
	for _, test := range []struct {
		data   string
		decode func([]byte) error
	}{
		{`{"version":1}`, func(data []byte) error { _, err := DecodeHostNetworks(data); return err }},
		{`{"version":1,"host_networks":[],"extra":true}`, func(data []byte) error { _, err := DecodeHostNetworks(data); return err }},
		{`{"version":1,"assignments":[]} {}`, func(data []byte) error { _, err := DecodeNetworkAssignments(data); return err }},
	} {
		if err := test.decode([]byte(test.data)); err == nil {
			t.Errorf("invalid JSON accepted: %s", test.data)
		}
	}
}
