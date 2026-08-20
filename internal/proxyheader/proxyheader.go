package proxyheader

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	headerFor   = "X-Forwarded-For"
	headerProto = "X-Forwarded-Proto"
)

type Trust struct {
	nets []netip.Prefix
}

func NewTrust(sources []string) (*Trust, error) {
	t := &Trust{}
	for _, raw := range sources {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if strings.Contains(s, "/") {
			p, err := netip.ParsePrefix(s)
			if err != nil {
				return nil, fmt.Errorf("parse trusted proxy CIDR %q: %w", s, err)
			}
			t.nets = append(t.nets, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(s)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy address %q: %w", s, err)
		}
		a = a.Unmap()
		t.nets = append(t.nets, netip.PrefixFrom(a, a.BitLen()))
	}
	return t, nil
}

func (t *Trust) Trusted(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, p := range t.nets {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (t *Trust) ClientIP(r *http.Request) (netip.Addr, bool) {
	peer, ok := PeerAddr(r.RemoteAddr)
	if !ok || !t.Trusted(peer) {
		return peer, ok
	}

	hops := forwardedHops(r.Header)
	for i := len(hops) - 1; i >= 0; i-- {
		if !t.Trusted(hops[i]) {
			return hops[i], true
		}
	}
	if len(hops) > 0 {
		return hops[0], true
	}
	return peer, true
}

func (t *Trust) Scheme(r *http.Request) string {
	direct := "http"
	if r.TLS != nil {
		direct = "https"
	}
	peer, ok := PeerAddr(r.RemoteAddr)
	if !ok || !t.Trusted(peer) {
		return direct
	}
	switch claimed := firstValue(r.Header.Get(headerProto)); claimed {
	case "http", "https":
		return claimed
	}
	return direct
}

func PeerAddr(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	a, err := netip.ParseAddr(strings.TrimSpace(host))
	if err != nil {
		return netip.Addr{}, false
	}
	return a.Unmap(), true
}

func forwardedHops(h http.Header) []netip.Addr {
	var hops []netip.Addr
	for _, line := range h.Values(headerFor) {
		for _, part := range strings.Split(line, ",") {
			a, err := netip.ParseAddr(strings.TrimSpace(part))
			if err != nil {
				continue
			}
			hops = append(hops, a.Unmap())
		}
	}
	return hops
}

func firstValue(header string) string {
	value, _, _ := strings.Cut(header, ",")
	return strings.ToLower(strings.TrimSpace(value))
}
