package shellpath

type entry struct {
	present bool
	link    bool
	target  string
	working bool
}

type action int

const (
	actCreate action = iota
	actKeep
	actReplace
	actRefuse
)

// A link that leads nowhere was left by a moved pwikit and is replaced freely.
// One that leads to a live program is somebody's working setup.
func decide(existing entry, exe string, force bool) action {
	switch {
	case !existing.present:
		return actCreate
	case existing.link && sameFile(existing.target, exe):
		return actKeep
	case existing.link && !existing.working:
		return actReplace
	case force:
		return actReplace
	}
	return actRefuse
}
