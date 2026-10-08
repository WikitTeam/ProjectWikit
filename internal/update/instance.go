package update

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const instanceFile = "instance.json"

type Instance struct {
	Root       string `json:"root"`
	Executable string `json:"executable"`
	Dev        uint64 `json:"dev"`
	Ino        uint64 `json:"ino"`
}

func ReadInstance(private string) (Instance, bool) {
	data, err := os.ReadFile(filepath.Join(private, instanceFile))
	if err != nil {
		return Instance{}, false
	}
	var in Instance
	if json.Unmarshal(data, &in) != nil || in.Root == "" {
		return Instance{}, false
	}
	return in, true
}
