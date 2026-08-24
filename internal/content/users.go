package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const UsersVersion = 1

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UsersFile struct {
	Version int    `json:"version"`
	Users   []User `json:"users"`
}

func EmptyUsers() UsersFile { return UsersFile{Version: UsersVersion, Users: []User{}} }

func DecodeUsers(data []byte) (UsersFile, error) {
	var file UsersFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return UsersFile{}, fmt.Errorf("users: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return UsersFile{}, fmt.Errorf("users: %w", err)
	}
	if file.Users == nil {
		return UsersFile{}, fmt.Errorf("users: users must be an array")
	}
	if err := ValidateUsers(file); err != nil {
		return UsersFile{}, err
	}
	return file, nil
}

func EncodeUsers(file UsersFile) ([]byte, error) {
	if file.Users == nil {
		file.Users = []User{}
	}
	if err := ValidateUsers(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("users: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateUsers(file UsersFile) error {
	if file.Version != UsersVersion {
		return fmt.Errorf("users: file version %d, want %d", file.Version, UsersVersion)
	}
	seen := make(map[string]struct{}, len(file.Users))
	for i, user := range file.Users {
		where := fmt.Sprintf("users: user %d", i+1)
		if !validUUID(user.ID) {
			return fmt.Errorf("%s has invalid UUID %q", where, user.ID)
		}
		if _, exists := seen[user.ID]; exists {
			return fmt.Errorf("%s duplicates UUID %q", where, user.ID)
		}
		seen[user.ID] = struct{}{}
		if !usernamePattern.MatchString(user.Username) {
			return fmt.Errorf("%s has invalid username %q", where, user.Username)
		}
		if strings.TrimSpace(user.Password) == "" {
			return fmt.Errorf("%s has an empty password", where)
		}
	}
	return nil
}

func ValidUsername(username string) bool { return usernamePattern.MatchString(username) }
