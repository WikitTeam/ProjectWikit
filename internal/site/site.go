package site

import (
	"net"
	"net/url"
	"strings"
)

type Site struct {
	Domain      string
	MediaDomain string
}

var MediaPrefixes = []string{
	"/local--files/",
	"/local--code/",
	"/local--html/",
	"/local--theme/",
}

type Action int

const (
	Serve Action = iota
	Redirect
)

type Decision struct {
	Action   Action
	Location string
	// Headers go on the response after the handler runs. A redirect carries
	// none.
	Headers map[string]string
}

var (
	crossOriginHeaders = map[string]string{"Access-Control-Allow-Origin": "*"}
	sameOriginHeaders  = map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}
)

func Decide(s Site, rawHost string, u *url.URL) Decision {
	onMediaHost := StripPort(rawHost) == s.MediaDomain
	wantsMedia := IsMediaPath(u.Path)

	if s.MediaDomain != "" && s.MediaDomain != s.Domain {
		switch {
		case onMediaHost && !wantsMedia:
			return Decision{Action: Redirect, Location: "//" + s.Domain + u.RequestURI()}
		case !onMediaHost && wantsMedia:
			return Decision{Action: Redirect, Location: "//" + s.MediaDomain + u.RequestURI()}
		}
	}

	if onMediaHost || (s.Domain == s.MediaDomain && wantsMedia) {
		return Decision{Action: Serve, Headers: crossOriginHeaders}
	}
	return Decision{Action: Serve, Headers: sameOriginHeaders}
}

func IsMediaPath(path string) bool {
	for _, prefix := range MediaPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func LookupHosts(rawHost, serverPort string) []string {
	var hosts []string
	if serverPort != "" && !strings.Contains(rawHost, ":") {
		hosts = append(hosts, rawHost+":"+serverPort)
	}
	return append(hosts, StripPort(rawHost))
}

func StripPort(rawHost string) string {
	if host, _, err := net.SplitHostPort(rawHost); err == nil {
		return host
	}
	return strings.Trim(rawHost, "[]")
}
