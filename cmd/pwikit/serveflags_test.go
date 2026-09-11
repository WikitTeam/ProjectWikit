package main

import (
	"slices"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/entry"
)

func resolved(t *testing.T, args []string, cfg config.File) (*serveOptions, entry.Mode) {
	t.Helper()
	o := newServeOptions()
	if err := o.fs.Parse(args); err != nil {
		t.Fatalf("Parse(%q) err = %v, want nil", args, err)
	}
	mode, err := o.resolve(cfg)
	if err != nil {
		t.Fatalf("resolve(%q) err = %v, want nil", args, err)
	}
	return o, mode
}

func TestResolveFlagBeatsEnvironmentBeatsFile(t *testing.T) {
	cfg := config.File{TLS: config.TLS{ACMEEmail: "file@example.com"}}

	o, _ := resolved(t, nil, cfg)
	if got := *o.acmeEmail; got != "file@example.com" {
		t.Errorf("acme-email with only the file = %q, want %q", got, "file@example.com")
	}

	t.Setenv(envACMEEmail, "env@example.com")
	o, _ = resolved(t, nil, cfg)
	if got := *o.acmeEmail; got != "env@example.com" {
		t.Errorf("acme-email with env and file = %q, want %q", got, "env@example.com")
	}

	o, _ = resolved(t, []string{"-acme-email", "flag@example.com"}, cfg)
	if got := *o.acmeEmail; got != "flag@example.com" {
		t.Errorf("acme-email with flag, env and file = %q, want %q", got, "flag@example.com")
	}
}

func TestResolveFallsBackToTheFlagDefault(t *testing.T) {
	o, mode := resolved(t, nil, config.File{})
	if mode != entry.Off {
		t.Errorf("mode = %q, want %q", mode, entry.Off)
	}
	if got := *o.listen; got != defaultListen {
		t.Errorf("listen = %q, want %q", got, defaultListen)
	}
	if got := *o.tlsListen; got != defaultTLSAddr {
		t.Errorf("tls-listen = %q, want %q", got, defaultTLSAddr)
	}
}

func TestResolveListensOnPort80OnceTLSIsOn(t *testing.T) {
	o, mode := resolved(t, nil, config.File{TLS: config.TLS{Mode: "auto"}})
	if mode != entry.Auto {
		t.Errorf("mode = %q, want %q", mode, entry.Auto)
	}
	if got := *o.listen; got != defaultTLSPlain {
		t.Errorf("listen = %q, want %q", got, defaultTLSPlain)
	}
}

func TestResolveKeepsAListenAddressFromTheFile(t *testing.T) {
	o, _ := resolved(t, nil, config.File{TLS: config.TLS{Mode: "auto"}, Server: config.Server{Listen: ":8080"}})
	if got := *o.listen; got != ":8080" {
		t.Errorf("listen = %q, want %q", got, ":8080")
	}
}

func TestResolveJoinsTrustedProxiesFromTheFile(t *testing.T) {
	o, _ := resolved(t, nil, config.File{Server: config.Server{TrustedProxies: []string{"10.0.0.0/8", "127.0.0.1"}}})
	if got := *o.trusted; got != "10.0.0.0/8,127.0.0.1" {
		t.Errorf("trusted-proxies = %q, want %q", got, "10.0.0.0/8,127.0.0.1")
	}
}

func TestResolveRefusesAnUnknownTLSMode(t *testing.T) {
	o := newServeOptions()
	if _, err := o.resolve(config.File{TLS: config.TLS{Mode: "maybe"}}); err == nil {
		t.Error("resolve(tls.mode = maybe) err = nil, want an error")
	}
}

func TestResolveIsAutomaticWhenNothingChoseHowPwikitIsReached(t *testing.T) {
	cases := []struct {
		name string
		args []string
		cfg  config.File
		want bool
	}{
		{"nothing set", nil, config.File{}, true},
		{"tls in the file", nil, config.File{TLS: config.TLS{Mode: "off"}}, false},
		{"listen on the command line", []string{"-listen", ":8080"}, config.File{}, false},
		{"listen in the file", nil, config.File{Server: config.Server{Listen: ":8080"}}, false},
		{"trusted proxies", nil, config.File{Server: config.Server{TrustedProxies: []string{"127.0.0.1"}}}, false},
		{"only an acme email", nil, config.File{TLS: config.TLS{ACMEEmail: "you@example.com"}}, true},
	}
	for _, c := range cases {
		o, _ := resolved(t, c.args, c.cfg)
		if o.automatic != c.want {
			t.Errorf("resolve(%s).automatic = %t, want %t", c.name, o.automatic, c.want)
		}
	}
}

func TestResolveAutomaticIsOffWithTLSFromTheEnvironment(t *testing.T) {
	t.Setenv(envTLS, "off")
	if o, _ := resolved(t, nil, config.File{}); o.automatic {
		t.Errorf("resolve with PWIKIT_TLS=off .automatic = true, want false")
	}
}

func TestPromoteTurnsHTTPSOnForAPublicDomain(t *testing.T) {
	o, _ := resolved(t, nil, config.File{})
	domain, ok := o.promote([]string{"localhost", "wiki.scp-wiki.cn"})
	if !ok || domain != "wiki.scp-wiki.cn" {
		t.Fatalf("promote() = %q, %t, want wiki.scp-wiki.cn, true", domain, ok)
	}
	if *o.listen != defaultTLSPlain {
		t.Errorf("listen after promote = %q, want %q", *o.listen, defaultTLSPlain)
	}
}

func TestPromoteLeavesALocalSiteOnPlainHTTP(t *testing.T) {
	o, _ := resolved(t, nil, config.File{})
	if _, ok := o.promote([]string{"localhost", "localhost:8080", "wiki.test"}); ok {
		t.Error("promote(local hosts) = true, want false")
	}
	if *o.listen != defaultListen {
		t.Errorf("listen = %q, want %q", *o.listen, defaultListen)
	}
}

func TestPromoteRespectsAnExplicitChoice(t *testing.T) {
	o, _ := resolved(t, nil, config.File{TLS: config.TLS{Mode: "off"}})
	if _, ok := o.promote([]string{"wiki.scp-wiki.cn"}); ok {
		t.Error("promote() with tls.mode = off = true, want false")
	}
}

func TestInstallAddressesPlanForHTTPSWhenAutomatic(t *testing.T) {
	o, mode := resolved(t, nil, config.File{})
	if got := exposedPorts(o.installAddresses(mode)); !slices.Equal(got, []int{80, 443}) {
		t.Errorf("exposedPorts(installAddresses) = %v, want [80 443]", got)
	}
	o, mode = resolved(t, []string{"-listen", "127.0.0.1:8080"}, config.File{})
	if got := exposedPorts(o.installAddresses(mode)); got != nil {
		t.Errorf("exposedPorts(installAddresses) behind a proxy = %v, want none", got)
	}
}

func TestDevListensOnlyOnThisMachine(t *testing.T) {
	cfg := config.File{TLS: config.TLS{Mode: "auto"}, Server: config.Server{Listen: ":80", TrustedProxies: []string{"10.0.0.0/8"}}}
	o, mode := resolved(t, []string{"-dev"}, cfg)
	if mode != entry.Off {
		t.Errorf("mode with -dev = %q, want %q", mode, entry.Off)
	}
	if *o.listen != defaultListen {
		t.Errorf("listen with -dev = %q, want %q", *o.listen, defaultListen)
	}
	if *o.trusted != "" {
		t.Errorf("trusted-proxies with -dev = %q, want empty", *o.trusted)
	}
	if o.automatic {
		t.Error("automatic with -dev = true, want false")
	}
	if _, ok := o.promote([]string{"wiki.scp-wiki.cn"}); ok {
		t.Error("promote() with -dev = true, want false")
	}
	if got := exposedPorts(o.installAddresses(mode)); got != nil {
		t.Errorf("exposedPorts(installAddresses) with -dev = %v, want none", got)
	}
}

func TestDevIgnoresTLSFromTheEnvironment(t *testing.T) {
	t.Setenv(envTLS, "auto")
	if _, mode := resolved(t, []string{"-dev"}, config.File{}); mode != entry.Off {
		t.Errorf("mode with -dev and PWIKIT_TLS=auto = %q, want %q", mode, entry.Off)
	}
}

func TestDevAcceptsAnotherLocalPort(t *testing.T) {
	cases := map[string]string{
		"9000":           "127.0.0.1:9000",
		":9000":          "127.0.0.1:9000",
		"127.0.0.1:9000": "127.0.0.1:9000",
		"localhost:9000": "localhost:9000",
		"[::1]:9000":     "[::1]:9000",
	}
	for given, want := range cases {
		o, _ := resolved(t, []string{"-dev", "-listen", given}, config.File{})
		if *o.listen != want {
			t.Errorf("listen with -dev -listen %s = %q, want %q", given, *o.listen, want)
		}
	}
}

func TestDevRefusesWhatOthersCouldReach(t *testing.T) {
	cases := [][]string{
		{"-dev", "-listen", "0.0.0.0:8080"},
		{"-dev", "-listen", "192.0.2.7:8080"},
		{"-dev", "-listen", "70000"},
		{"-dev", "-listen", "web"},
		{"-dev", "-tls", "auto"},
	}
	for _, args := range cases {
		o := newServeOptions()
		if err := o.fs.Parse(args); err != nil {
			t.Fatalf("Parse(%q) err = %v, want nil", args, err)
		}
		if _, err := o.resolve(config.File{}); err == nil {
			t.Errorf("resolve(%q) err = nil, want an error", args)
		}
	}
}

func TestDevStillReadsTheDatabaseFromTheFile(t *testing.T) {
	o, _ := resolved(t, []string{"-dev"}, config.File{Database: "postgres://dev@127.0.0.1/wiki"})
	if *o.database != "postgres://dev@127.0.0.1/wiki" {
		t.Errorf("database with -dev = %q, want the file's", *o.database)
	}
}

func TestExposedPortsSkipsLoopback(t *testing.T) {
	cases := []struct {
		addresses []string
		want      []int
	}{
		{[]string{"127.0.0.1:8080"}, nil},
		{[]string{"localhost:8080", "[::1]:8443"}, nil},
		{[]string{":80", ":443"}, []int{80, 443}},
		{[]string{"0.0.0.0:8080", "192.0.2.7:8080"}, []int{8080}},
		{[]string{"not an address"}, nil},
	}
	for _, c := range cases {
		if got := exposedPorts(c.addresses); !slices.Equal(got, c.want) {
			t.Errorf("exposedPorts(%q) = %v, want %v", c.addresses, got, c.want)
		}
	}
}

func TestMailConfigReadsTheFileUnderTheEnvironment(t *testing.T) {
	yes := true
	file := config.Mail{Host: "smtp.file.test", Port: 587, UseTLS: &yes, From: "wiki@file.test"}

	got := mailConfig(file)
	if got.Host != "smtp.file.test" || got.Port != "587" || !got.UseTLS || got.From != "wiki@file.test" {
		t.Errorf("mailConfig(file) = %+v, want host, port 587, TLS and from taken from the file", got)
	}

	t.Setenv(envMailHost, "smtp.env.test")
	t.Setenv(envMailTLS, "false")
	got = mailConfig(file)
	if got.Host != "smtp.env.test" {
		t.Errorf("mailConfig(file).Host with env = %q, want %q", got.Host, "smtp.env.test")
	}
	if got.UseTLS {
		t.Errorf("mailConfig(file).UseTLS with EMAIL_USE_TLS=false = true, want false")
	}
}

func TestMailConfigConsoleSendsNothing(t *testing.T) {
	if got := mailConfig(config.Mail{Engine: config.EngineConsole, Host: "smtp.file.test"}); got.Host != "" {
		t.Errorf("mailConfig(console).Host = %q, want empty", got.Host)
	}
}

func TestMailConfigDefaultPort(t *testing.T) {
	if got := mailConfig(config.Mail{}).Port; got != defaultMailPort {
		t.Errorf("mailConfig(empty).Port = %q, want %q", got, defaultMailPort)
	}
}
