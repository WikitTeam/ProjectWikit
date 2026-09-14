//go:build cgo && !nocgo

package main

import (
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	ftml "github.com/WikitTeam/ProjectWikit/internal/renderer/cgo"
)

func newRenderer(sidecarPath string) (renderer.Renderer, func(), error) {
	if sidecarPath != "" {
		return newSidecarEngine(sidecarPath)
	}
	return ftml.New(), func() {}, nil
}
