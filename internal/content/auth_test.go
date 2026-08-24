package content

import "testing"

func TestUsersAndTerminalAccessRoundTrip(t *testing.T) {
	secondUUID := "123e4567-e89b-42d3-a456-426614174001"
	users := UsersFile{Version: UsersVersion, Users: []User{{ID: testUUID, Username: "operator", Password: "fictional password"}}}
	data, err := EncodeUsers(users)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeUsers(data)
	if err != nil || len(decoded.Users) != 1 || decoded.Users[0] != users.Users[0] {
		t.Fatalf("user round trip = %+v, %v", decoded, err)
	}

	access := TerminalAccessFile{Version: TerminalAccessVersion, Access: []TerminalAccess{{UserID: testUUID, TerminalID: secondUUID}}}
	data, err = EncodeTerminalAccess(access)
	if err != nil {
		t.Fatal(err)
	}
	decodedAccess, err := DecodeTerminalAccess(data)
	if err != nil || len(decodedAccess.Access) != 1 || decodedAccess.Access[0] != access.Access[0] {
		t.Fatalf("access round trip = %+v, %v", decodedAccess, err)
	}
}

func TestUsersAndTerminalAccessRejectInvalidContent(t *testing.T) {
	secondUUID := "123e4567-e89b-42d3-a456-426614174001"
	tests := []struct {
		name string
		err  error
	}{
		{"user version", ValidateUsers(UsersFile{Version: 2, Users: []User{}})},
		{"user UUID", ValidateUsers(UsersFile{Version: UsersVersion, Users: []User{{ID: "bad", Username: "operator", Password: "fictional"}}})},
		{"user name", ValidateUsers(UsersFile{Version: UsersVersion, Users: []User{{ID: testUUID, Username: "Bad User", Password: "fictional"}}})},
		{"user password", ValidateUsers(UsersFile{Version: UsersVersion, Users: []User{{ID: testUUID, Username: "operator", Password: " "}}})},
		{"duplicate user UUID", ValidateUsers(UsersFile{Version: UsersVersion, Users: []User{{ID: testUUID, Username: "one", Password: "a"}, {ID: testUUID, Username: "two", Password: "b"}}})},
		{"access version", ValidateTerminalAccess(TerminalAccessFile{Version: 2, Access: []TerminalAccess{}})},
		{"access user UUID", ValidateTerminalAccess(TerminalAccessFile{Version: TerminalAccessVersion, Access: []TerminalAccess{{UserID: "bad", TerminalID: secondUUID}}})},
		{"access terminal UUID", ValidateTerminalAccess(TerminalAccessFile{Version: TerminalAccessVersion, Access: []TerminalAccess{{UserID: testUUID, TerminalID: "bad"}}})},
		{"duplicate access", ValidateTerminalAccess(TerminalAccessFile{Version: TerminalAccessVersion, Access: []TerminalAccess{{UserID: testUUID, TerminalID: secondUUID}, {UserID: testUUID, TerminalID: secondUUID}}})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("invalid content was accepted")
			}
		})
	}
}

func TestUsersAndTerminalAccessDecodersAreStrict(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"version":1}`),
		[]byte(`{"version":1,"users":[],"extra":true}`),
		[]byte(`{"version":1,"users":[]} {}`),
	} {
		if _, err := DecodeUsers(data); err == nil {
			t.Fatalf("invalid users JSON accepted: %s", data)
		}
	}
	for _, data := range [][]byte{
		[]byte(`{"version":1}`),
		[]byte(`{"version":1,"access":[],"extra":true}`),
		[]byte(`{"version":1,"access":[]} {}`),
	} {
		if _, err := DecodeTerminalAccess(data); err == nil {
			t.Fatalf("invalid access JSON accepted: %s", data)
		}
	}
}
