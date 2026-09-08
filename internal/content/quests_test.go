package content

import "testing"

func TestQuestCatalogCompatibilityAndIDs(t *testing.T) {
	c, err := DecodeCatalogs(nil)
	if err != nil || len(c.Quests.Quests) != 0 || c.Quests.Version != 1 {
		t.Fatal("legacy catalog failed", err)
	}
	for _, raw := range []string{`{"version":2}`, `{"version":1,"quests":[{"id":"q"},{"id":"q"}]}`, `{"version":1,"quests":[{"id":"q","steps":[{"id":"s"},{"id":"s"}]}]}`, `{"version":1,"quests":[{"id":"q?bad"}]}`} {
		if _, err := DecodeQuests([]byte(raw)); err == nil {
			t.Fatal("accepted invalid catalog", raw)
		}
	}
}
