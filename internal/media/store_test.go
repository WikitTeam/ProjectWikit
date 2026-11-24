package media

import "github.com/WikitTeam/ProjectWikit/internal/storage"

func localSet(root string) *storage.Set {
	return storage.Single(&storage.Backend{Store: storage.NewLocal(root), Mode: storage.ModeLocal})
}
