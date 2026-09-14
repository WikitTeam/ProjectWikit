package main

import (
	"errors"
	"fmt"
	"os"
)

const usage = `Usage: go run ./tools/release <build|manifest> [options]

  build     build the release package for this machine's platform into -out
  manifest  write SHA256SUMS and latest.json for the packages in -dir

Run from the repository root. Add -h after a subcommand to list its options.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "release: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return errors.New("missing subcommand")
	}
	switch args[0] {
	case "build":
		return buildCommand(args[1:])
	case "manifest":
		return manifestCommand(args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stderr, usage)
		return nil
	}
	fmt.Fprint(os.Stderr, usage)
	return fmt.Errorf("unknown subcommand %q", args[0])
}

func packageBase(version, goos, goarch string) string {
	return "pwikit-" + version + "-" + goos + "-" + goarch
}

func packageFile(version, goos, goarch string) string {
	if goos == "windows" {
		return packageBase(version, goos, goarch) + ".zip"
	}
	return packageBase(version, goos, goarch) + ".tar.gz"
}

func executableName(goos string) string {
	if goos == "windows" {
		return "pwikit.exe"
	}
	return "pwikit"
}
