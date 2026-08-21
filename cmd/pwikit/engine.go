package main

import (
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

func newSidecarEngine(path string) (renderer.Renderer, func(), error) {
	r, err := sidecar.New(path)
	if err != nil {
		return nil, nil, err
	}
	return r, func() { r.Close() }, nil
}
