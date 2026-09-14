package roles

import (
	"os"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

// FileIcons reads role icons out of the media root. The icon column holds a
// path chosen in the admin, so it goes through Resolve rather than a plain join.
func FileIcons(root string) IconLoader {
	return func(name string) (string, error) {
		path, err := paths.Resolve(root, name)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
