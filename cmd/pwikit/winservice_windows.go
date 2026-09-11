package main

import (
	"context"
	"errors"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"

	"github.com/WikitTeam/ProjectWikit/internal/service"
)

func runAsService(args []string) (bool, error) {
	isService, err := svc.IsWindowsService()
	if err != nil || !isService {
		return false, err
	}
	return true, svc.Run(service.DefaultName, &windowsService{args: args})
}

type windowsService struct {
	args []string
}

func (w *windowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		if len(w.args) == 0 || w.args[0] != "serve" {
			done <- errors.New("a pwikit service can only run pwikit serve")
			return
		}
		done <- serve(ctx, w.args[1:])
	}()

	accepts := svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPreShutdown
	status <- svc.Status{State: svc.Running, Accepts: accepts}
	for {
		select {
		case err := <-done:
			if err != nil {
				report(err)
				// A nonzero exit code is what makes the recovery actions restart the service.
				return true, 1
			}
			return false, 0
		case r := <-requests:
			switch r.Cmd {
			case svc.Interrogate:
				status <- r.CurrentStatus
			case svc.Stop, svc.Shutdown, svc.PreShutdown:
				status <- svc.Status{State: svc.StopPending, WaitHint: uint32(service.StopTimeout * 1000)}
				cancel()
			}
		}
	}
}

func report(err error) {
	log, openErr := eventlog.Open(service.EventSource)
	if openErr != nil {
		return
	}
	defer log.Close()
	log.Error(1, "pwikit stopped: "+err.Error())
}
