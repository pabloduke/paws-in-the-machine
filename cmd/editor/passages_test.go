package main

import (
	"net/url"
	"path"
	"strings"
	"testing"
)

func TestPassageEditorAndAPI(t *testing.T) {
	e := newContentsTestEditor(t)
	_, loc, room := e.oneRoomWorld(t)
	target := entityURL("room", room)
	second := drillSubmit(t, e, entityURL("location", loc), "create-child", url.Values{"child_name": {"Neighbor"}, "child_description": {"(Placeholder)"}, "x": {"1"}, "y": {"0"}, "z": {"0"}})
	other := path.Base(second)
	drillCheck(t, e, target, "Entrance and exits", "Neighbor", "Block east path")
	root := "/api/v1/worlds/main"
	rev := apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag")
	body := `{"action":"set-passage","other_id":"` + other + `","blocked":"true","expected_blocked":"false"}`
	r := apiRequest(t, e.handler, "POST", root+"/rooms/"+room+"/actions", rev, body)
	if r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	drillCheck(t, e, target, "Reopen east path")
	drillCheck(t, e, second, "Reopen west path")
	if apiRequest(t, e.handler, "POST", root+"/rooms/"+room+"/actions", rev, body).Code != 409 {
		t.Fatal("stale API write accepted")
	}
	failed := postForm(t, e.handler, target+"/work", url.Values{"action": {"set-passage"}, "other_id": {other}, "blocked": {"true"}, "expected_blocked": {"false"}}, "http://example.com", false)
	if !strings.Contains(failed.Body.String(), "Passage changed") {
		t.Fatal(failed.Body.String())
	}
	failed = postForm(t, e.handler, target+"/work", url.Values{"action": {"unplace"}, "parent_id": {loc}}, "http://example.com", false)
	// Moving via the common placement screen must guard walls as well.
	if failed.Code == 303 || !strings.Contains(failed.Body.String(), "blocked passages") {
		t.Fatal("unplace guard missing", failed.Body.String())
	}
	drillSubmit(t, e, second, "set-passage", url.Values{"other_id": {room}, "blocked": {"false"}, "expected_blocked": {"true"}})
	drillCheck(t, e, target, "Block east path")
}
func TestEntranceChoicesAcrossFloors(t *testing.T) {
	e := newContentsTestEditor(t)
	_, loc, room := e.oneRoomWorld(t)
	target := entityURL("location", loc)
	upper := drillSubmit(t, e, target, "create-child", url.Values{"child_name": {"Upper Room"}, "child_description": {"(Placeholder)"}, "x": {"0"}, "y": {"0"}, "z": {"1"}})
	drillCheck(t, e, target+"?z=0", "Upper Room (floor 1)", "Entrance Room", "Save entrance")
	drillSubmit(t, e, target, "entry", url.Values{"entry_id": {room}})
	drillCheck(t, e, entityURL("room", room), "This is the entrance Room")
	drillCheck(t, e, upper, "Set the entrance on the parent Location")
}
