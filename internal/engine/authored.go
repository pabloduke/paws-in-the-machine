package engine

// RoomBoundary prevents a containing exterior cell from reaching into an
// interior Room. It does not change scope when the player occupies that Room.
type RoomBoundary struct{}

func (RoomBoundary) Handle(*World, *Entity, Command) (string, bool) { return "", false }

// ShortDescription supplements the targetable name in the room's item list.
type ShortDescription struct{ Text string }

func (ShortDescription) Handle(*World, *Entity, Command) (string, bool) { return "", false }

func Listing(w *World, e *Entity) string {
	name := DisplayName(w, e)
	if d, ok := Part[ShortDescription](e); ok && d.Text != "" {
		return name + " — " + d.Text
	}
	return name
}
