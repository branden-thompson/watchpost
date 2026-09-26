package geo

// arrangement.go — how the regions stand beside one another (0.18.0 D-77,
// UAT-1 U1-40): the map is bound to one region at a time (D-28), and the
// listener moves between them by number or across an edge.
//
// THE LAYOUT IS THE HUM LEAD'S, as drawn:
//
//	                          [ Alaska ]
//	[ Guam ] [ Samoa ] [ Hawaii ] [ Contiguous US ] [ Caribbean ]
//
// West and east run along the lower row; Alaska is north of Hawaii and of the
// contiguous United States, and south of Alaska is the contiguous United
// States. An edge with nothing beyond it leads nowhere.

// Direction is a way off a region's edge.
type Direction int

// The four edges.
const (
	North Direction = iota
	South
	East
	West
)

// String is the direction's name, for a test's message.
func (d Direction) String() string {
	return [...]string{"north", "south", "east", "west"}[d]
}

// placement is a region's number and its neighbours by direction.
type placement struct {
	number int
	next   map[Direction]string
}

// arrangement is every region's place in the layout.
var arrangement = map[string]placement{
	RegionContiguous: {1, map[Direction]string{West: RegionHawaii, East: RegionCaribbean, North: RegionAlaska}},
	RegionAlaska:     {2, map[Direction]string{South: RegionContiguous}},
	RegionHawaii:     {3, map[Direction]string{West: RegionSamoa, East: RegionContiguous, North: RegionAlaska}},
	RegionCaribbean:  {4, map[Direction]string{West: RegionContiguous}},
	RegionSamoa:      {5, map[Direction]string{West: RegionMarianas, East: RegionHawaii}},
	RegionMarianas:   {6, map[Direction]string{East: RegionSamoa}},
}

// Neighbour is the region past a region's edge, and false where the layout
// ends.
func Neighbour(name string, dir Direction) (Region, bool) {
	next, ok := arrangement[name].next[dir]
	if !ok {
		return Region{}, false
	}
	return regionNamed(next)
}

// RegionNumbered is the region a number snaps to, 1 to 6.
func RegionNumbered(n int) (Region, bool) {
	for name, p := range arrangement {
		if p.number == n {
			return regionNamed(name)
		}
	}
	return Region{}, false
}

// NumberOf is a region's number, and 0 for a name that is not a region.
func NumberOf(name string) int { return arrangement[name].number }

// regionNamed is the region of a name.
func regionNamed(name string) (Region, bool) {
	for _, r := range regions {
		if r.Name == name {
			return r, true
		}
	}
	return Region{}, false
}
