package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/service"
)

func serviceUsage() {
	fmt.Fprint(os.Stderr, `Usage: pwikit service <install|uninstall|start|stop|status|print> [options] [-- serve options]

  install    start pwikit whenever the machine boots, and start it now
  uninstall  stop pwikit and no longer start it at boot
  start      start the installed service
  stop       stop the installed service
  status     show whether the installed service is running
  print      show what install would register, without registering it

Options:
  -name      name the service is registered under; defaults to pwikit
  -user      account the service runs as on Linux and macOS; defaults to the one sudo was run from
  -data-dir  state directory; defaults to the directory holding the executable

Anything after -- is handed to pwikit serve, for example:
  pwikit service install -- -tls=auto -acme-email you@example.com

On Linux and on a Mac that should start it at boot, run install with sudo.
On Windows, run it from a terminal opened with Run as administrator.
`)
}

func serviceCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "install", "uninstall", "start", "stop", "status", "print":
	default:
		serviceUsage()
		return errors.New("unknown service subcommand")
	}

	fs := flag.NewFlagSet("service "+sub, flag.ContinueOnError)
	name := fs.String("name", service.DefaultName, "name the service is registered under")
	account := fs.String("user", "", "account the service runs as on Linux and macOS")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := service.ValidName(*name); err != nil {
		return err
	}

	switch sub {
	case "uninstall":
		return service.Uninstall(*name)
	case "start":
		return service.Start(*name)
	case "stop":
		return service.Stop(*name)
	case "status":
		return service.Status(*name)
	}

	spec, err := serviceSpec(*name, *account, *dataDir, fs.Args())
	if err != nil {
		return err
	}
	if sub == "print" {
		text, err := service.Preview(spec)
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	}
	if err := service.Install(spec); err != nil {
		return err
	}
	fmt.Printf("installed %s; it starts now and whenever the machine boots\n", spec.Name)
	fmt.Println(afterInstall(spec))
	return nil
}

func serviceSpec(name, account, dataDir string, extra []string) (service.Spec, error) {
	p, err := paths.New(dataDir)
	if err != nil {
		return service.Spec{}, err
	}
	if p.Source() == paths.SourceGoRun {
		return service.Spec{}, errors.New("go run leaves the program in a temporary directory; build pwikit and install that instead")
	}
	exe, err := os.Executable()
	if err != nil {
		return service.Spec{}, err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	args := []string{"serve", "-data-dir", p.Root()}
	spec := service.Spec{Name: name, Executable: exe, Root: p.Root(), User: account}
	if runtime.GOOS != "linux" {
		spec.LogFile = filepath.Join(p.Logs(), "pwikit.log")
		args = append(args, "-log-file", spec.LogFile)
	}
	if runtime.GOOS == "darwin" {
		spec.CrashFile = filepath.Join(p.Logs(), "pwikit-stderr.log")
	}
	spec.Args = append(args, extra...)

	opts := newServeOptions()
	if err := opts.fs.Parse(extra); err != nil {
		return service.Spec{}, fmt.Errorf("options for serve: %w", err)
	}
	if opts.fs.NArg() > 0 {
		return service.Spec{}, fmt.Errorf("options for serve: unexpected %q", opts.fs.Arg(0))
	}
	cfg, err := config.Load(p.Config())
	if err != nil {
		return service.Spec{}, err
	}
	mode, err := opts.resolve(cfg)
	if err != nil {
		return service.Spec{}, err
	}
	spec.Ports = exposedPorts(opts.installAddresses(mode))
	return spec, nil
}

func afterInstall(spec service.Spec) string {
	if runtime.GOOS == "linux" {
		return "  follow its log with: journalctl -u " + spec.Name + " -f"
	}
	return "  its log is " + spec.LogFile
}
