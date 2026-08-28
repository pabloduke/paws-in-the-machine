package main

import (
	"errors"
	"path/filepath"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errLocationAssignmentNotFound = errors.New("location assignment not found")
var errRoomAssignmentNotFound = errors.New("room assignment not found")

// Containment: which Hub owns a Location, and which Location owns a
// Room. Coordinates live in the placement catalogs, never here — a
// child may be owned and unplaced, and unplacing preserves ownership.

type locationAssignmentStore struct {
	*relationStore[gamecontent.LocationAssignment]
}

func newLocationAssignmentStore(contentDir string) *locationAssignmentStore {
	return &locationAssignmentStore{&relationStore[gamecontent.LocationAssignment]{
		path:  filepath.Join(contentDir, "location_assignments.json"),
		label: "location assignments",
		key:   func(a gamecontent.LocationAssignment) string { return a.LocationID },
		decode: func(data []byte) ([]gamecontent.LocationAssignment, error) {
			file, err := gamecontent.DecodeLocationAssignments(data)
			return file.Assignments, err
		},
		encode: func(records []gamecontent.LocationAssignment) ([]byte, error) {
			return gamecontent.EncodeLocationAssignments(gamecontent.LocationAssignmentsFile{
				Version: gamecontent.LocationAssignmentsVersion, Assignments: records,
			})
		},
	}}
}

func (s *locationAssignmentStore) Get(locationID string) (gamecontent.LocationAssignment, error) {
	assignment, err := s.relationStore.Get(locationID)
	if errors.Is(err, errRelationNotFound) {
		return assignment, errLocationAssignmentNotFound
	}
	return assignment, err
}

func (s *locationAssignmentStore) Assign(locationID, hubID string) (gamecontent.LocationAssignment, error) {
	return s.Put(gamecontent.LocationAssignment{LocationID: locationID, HubID: hubID})
}

func (s *locationAssignmentStore) Unassign(locationID string) (gamecontent.LocationAssignment, error) {
	assignment, err := s.Delete(locationID)
	if errors.Is(err, errRelationNotFound) {
		return assignment, errLocationAssignmentNotFound
	}
	return assignment, err
}

type roomAssignmentStore struct {
	*relationStore[gamecontent.RoomAssignment]
}

func newRoomAssignmentStore(contentDir string) *roomAssignmentStore {
	return &roomAssignmentStore{&relationStore[gamecontent.RoomAssignment]{
		path:  filepath.Join(contentDir, "room_assignments.json"),
		label: "room assignments",
		key:   func(a gamecontent.RoomAssignment) string { return a.RoomID },
		decode: func(data []byte) ([]gamecontent.RoomAssignment, error) {
			file, err := gamecontent.DecodeRoomAssignments(data)
			return file.Assignments, err
		},
		encode: func(records []gamecontent.RoomAssignment) ([]byte, error) {
			return gamecontent.EncodeRoomAssignments(gamecontent.RoomAssignmentsFile{
				Version: gamecontent.RoomAssignmentsVersion, Assignments: records,
			})
		},
	}}
}

func (s *roomAssignmentStore) Get(roomID string) (gamecontent.RoomAssignment, error) {
	assignment, err := s.relationStore.Get(roomID)
	if errors.Is(err, errRelationNotFound) {
		return assignment, errRoomAssignmentNotFound
	}
	return assignment, err
}

func (s *roomAssignmentStore) Assign(roomID, locationID string) (gamecontent.RoomAssignment, error) {
	return s.Put(gamecontent.RoomAssignment{RoomID: roomID, LocationID: locationID})
}

func (s *roomAssignmentStore) Unassign(roomID string) (gamecontent.RoomAssignment, error) {
	assignment, err := s.Delete(roomID)
	if errors.Is(err, errRelationNotFound) {
		return assignment, errRoomAssignmentNotFound
	}
	return assignment, err
}
