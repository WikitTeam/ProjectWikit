package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	toolFirewalld = "firewalld"
	toolUFW       = "ufw"
)

func openPorts(ports []int) Firewall {
	if len(ports) == 0 {
		return Firewall{}
	}
	switch {
	case quiet("firewall-cmd", "--state") == nil:
		opened := Firewall{Tool: toolFirewalld}
		for _, port := range ports {
			spec := strconv.Itoa(port) + "/tcp"
			if quiet("firewall-cmd", "--permanent", "--query-port="+spec) == nil {
				continue
			}
			if err := runTool("firewall-cmd", "--permanent", "--add-port="+spec); err != nil {
				warnFirewall(port, err)
				continue
			}
			opened.Ports = append(opened.Ports, port)
		}
		if len(opened.Ports) > 0 {
			runTool("firewall-cmd", "--reload")
		}
		return opened
	case ufwActive():
		opened := Firewall{Tool: toolUFW}
		for _, port := range ports {
			if ufwAllows(port) {
				continue
			}
			if err := runTool("ufw", "allow", strconv.Itoa(port)+"/tcp"); err != nil {
				warnFirewall(port, err)
				continue
			}
			opened.Ports = append(opened.Ports, port)
		}
		return opened
	}
	return Firewall{}
}

func closePorts(f Firewall) {
	switch f.Tool {
	case toolFirewalld:
		for _, port := range f.Ports {
			runTool("firewall-cmd", "--permanent", "--remove-port="+strconv.Itoa(port)+"/tcp")
		}
		if len(f.Ports) > 0 {
			runTool("firewall-cmd", "--reload")
		}
	case toolUFW:
		for _, port := range f.Ports {
			runTool("ufw", "delete", "allow", strconv.Itoa(port)+"/tcp")
		}
	}
}

func quiet(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func ufwActive() bool {
	out, err := exec.Command("ufw", "status").Output()
	return err == nil && strings.HasPrefix(strings.TrimSpace(string(out)), "Status: active")
}

func ufwAllows(port int) bool {
	out, err := exec.Command("ufw", "status").Output()
	if err != nil {
		return false
	}
	want := strconv.Itoa(port)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "ALLOW" {
			continue
		}
		if fields[0] == want || fields[0] == want+"/tcp" {
			return true
		}
	}
	return false
}

func warnFirewall(port int, err error) {
	fmt.Fprintf(os.Stderr, "could not open port %d in the firewall: %v; visitors from other machines stay blocked until it is opened\n", port, err)
}
