//go:build !cgo || nocgo

package main

import (
	"errors"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func newRenderer(sidecarPath string) (renderer.Renderer, func(), error) {
	if sidecarPath == "" {
		return nil, nil, errors.New("this build has no ftml linked in; pass -sidecar or set " + envSidecar)
	}
	return newSidecarEngine(sidecarPath)
}
