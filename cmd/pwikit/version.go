package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/pgbundle"
	"github.com/WikitTeam/ProjectWikit/internal/version"
)

func printVersion() error {
	fmt.Printf("pwikit %s\n", version.String())
	if commit := version.Commit(); commit != "" {
		fmt.Printf("commit      %s\n", commit)
	}
	if at := version.BuiltFrom(); !at.IsZero() {
		fmt.Printf("committed   %s\n", at.UTC().Format(time.RFC3339))
	}
	fmt.Printf("platform    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	if pgbundle.Embedded() {
		fmt.Printf("postgresql  %s, bundled\n", pgbundle.Version)
	} else {
		fmt.Println("postgresql  not bundled")
	}
	fmt.Printf("go          %s\n", runtime.Version())
	return nil
}
