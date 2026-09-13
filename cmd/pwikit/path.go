package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/shellpath"
)

func pathUsage() {
	fmt.Fprint(os.Stderr, `Usage: pwikit path <install|uninstall|status> [options]

  install    make pwikit runnable by name from any directory
  uninstall  undo install
  status     show where the pwikit command is reachable from

Options:
  -dir    directory to put the command in on Linux and macOS; defaults to
          /usr/local/bin, or ~/.local/bin when that cannot be written
  -force  replace a command of the same name that leads somewhere else

On Windows, install adds the directory holding pwikit to your own Path.
`)
}

func pathCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "install", "uninstall", "status":
	default:
		pathUsage()
		return errors.New("unknown path subcommand")
	}

	fs := flag.NewFlagSet("path "+sub, flag.ContinueOnError)
	dir := fs.String("dir", "", "directory to put the command in on Linux and macOS")
	force := fs.Bool("force", false, "replace a command of the same name that leads somewhere else")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	exe, err := runnableExecutable()
	if err != nil {
		return err
	}
	switch sub {
	case "install":
		return installPath(exe, *dir, *force)
	case "uninstall":
		return uninstallPath(exe, *dir)
	}
	return pathStatus(exe, *dir)
}

// A program run from go run sits in a temporary directory that is gone when it
// exits, so nothing may be pointed at it.
func runnableExecutable() (string, error) {
	p, err := paths.New("")
	if err != nil {
		return "", err
	}
	if p.Source() == paths.SourceGoRun {
		return "", errors.New("go run leaves the program in a temporary directory; build pwikit and run that instead")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

func installPath(exe, dir string, force bool) error {
	place, outcome, err := shellpath.Install(exe, dir, force)
	if err != nil {
		return err
	}
	switch {
	case outcome == shellpath.Unchanged:
		fmt.Printf("pwikit is already reachable through %s\n", place.Path)
	case runtime.GOOS == "windows":
		fmt.Printf("added %s to your Path\n", place.Path)
	case outcome == shellpath.Replaced:
		fmt.Printf("pointed %s at %s, replacing an older one\n", place.Path, exe)
	default:
		fmt.Printf("pointed %s at %s\n", place.Path, exe)
	}
	printReach(place)
	return nil
}

func uninstallPath(exe, dir string) error {
	place, outcome, err := shellpath.Uninstall(exe, dir)
	if err != nil {
		return err
	}
	if outcome == shellpath.Absent {
		fmt.Println("pwikit was not on the PATH through this command")
		return nil
	}
	fmt.Println(removedMessage(place))
	return nil
}

func removedMessage(place shellpath.Place) string {
	if runtime.GOOS == "windows" {
		return "removed " + place.Path + " from your Path"
	}
	return "removed " + place.Path
}

func pathStatus(exe, dir string) error {
	places, err := shellpath.Status(exe, dir)
	if err != nil {
		return err
	}
	if len(places) == 0 {
		fmt.Println("pwikit is not on the PATH; run ./pwikit path install from its directory")
		return nil
	}
	for _, place := range places {
		state := "leads to this pwikit"
		switch {
		case !place.Working:
			state = "leads nowhere; run ./pwikit path install from the new directory"
		case !place.Ours:
			state = "leads to another pwikit at " + place.Target
		}
		fmt.Printf("%s: %s\n", place.Path, state)
		printReach(place)
	}
	return nil
}

func printReach(place shellpath.Place) {
	if runtime.GOOS == "windows" {
		fmt.Println("  open a new terminal for the change to take effect")
		return
	}
	if !place.OnPath {
		fmt.Printf("  %s is not on your PATH; add it in your shell profile\n", filepath.Dir(place.Path))
	}
}

// Setting up the service is when pwikit is being put in place for good, so the
// command goes on the PATH then too. A failure here leaves the service working.
func servicePath(exe string, install bool) {
	if install {
		place, _, err := shellpath.Install(exe, "", false)
		if err != nil {
			fmt.Printf("  could not put pwikit on the PATH: %v\n  run pwikit path install to try again\n", err)
			return
		}
		fmt.Printf("  pwikit can now be run by name through %s\n", place.Path)
		printReach(place)
		return
	}
	place, outcome, err := shellpath.Uninstall(exe, "")
	if err != nil {
		fmt.Printf("  could not take pwikit off the PATH: %v\n", err)
		return
	}
	if outcome == shellpath.Removed {
		fmt.Println("  " + removedMessage(place))
	}
}
