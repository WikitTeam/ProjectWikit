package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/entry"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type serveOptions struct {
	fs            *flag.FlagSet
	listen        *string
	dataDir       *string
	trusted       *string
	staticDir     *string
	database      *string
	secret        *string
	sidecar       *string
	noMigrate     *bool
	uploadLimit   *string
	storageLimit  *string
	tlsMode       *string
	tlsListen     *string
	tlsCert       *string
	tlsKey        *string
	acmeEmail     *string
	acmeDirectory *string
	logFile       *string
	dev           *bool

	automatic bool
}

func newServeOptions() *serveOptions {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	return &serveOptions{
		fs:            fs,
		listen:        fs.String("listen", defaultListen, "listen address; "+defaultTLSPlain+" once HTTPS is on"),
		dataDir:       fs.String("data-dir", "", "state directory; defaults to the directory holding the executable"),
		trusted:       fs.String("trusted-proxies", "", "trusted reverse proxy addresses or CIDRs, comma separated; empty trusts no X-Forwarded-* header"),
		staticDir:     fs.String("static-dir", "", "directory holding the frontend asset bundle"),
		database:      fs.String("database", "", "PostgreSQL connection string; empty runs the bundled PostgreSQL"),
		secret:        fs.String("secret-key", "", "key the session cookie is signed with; defaults to one pwikit makes and keeps in secrets/"+sessionKeyFile),
		sidecar:       fs.String("sidecar", "", "path to the ftml sidecar binary; without it the linked-in ftml is used"),
		noMigrate:     fs.Bool("no-migrate", false, "start without applying pending schema migrations"),
		uploadLimit:   fs.String("upload-limit", "0", "size the files still attached to pages may reach, such as 4GB; 0 for no ceiling"),
		storageLimit:  fs.String("storage-limit", "0", "size every file on disk may reach, deleted ones counted; 0 for no ceiling"),
		tlsMode:       fs.String("tls", string(entry.Off), "off to serve plain HTTP behind a proxy, file to use a supplied certificate, auto to obtain one over ACME; left unset, auto once a site is bound to a public domain"),
		tlsListen:     fs.String("tls-listen", defaultTLSAddr, "listen address for HTTPS"),
		tlsCert:       fs.String("tls-cert", "", "certificate chain in PEM form, for -tls=file"),
		tlsKey:        fs.String("tls-key", "", "private key in PEM form, for -tls=file"),
		acmeEmail:     fs.String("acme-email", "", "address the certificate authority sends expiry warnings to"),
		acmeDirectory: fs.String("acme-directory", "", "ACME directory URL; empty uses Let's Encrypt"),
		logFile:       fs.String("log-file", "", "file log lines are appended to; empty writes them to standard error"),
		dev:           fs.Bool("dev", false, "development mode; plain HTTP that only this machine can reach, whatever the sites and settings say; -listen 9000 picks another port"),
	}
}

func (o *serveOptions) resolve(cfg config.File) (entry.Mode, error) {
	if *o.dev {
		return o.resolveDev(cfg)
	}
	pick := func(name, env, file string) string {
		return setting(o.fs, name, env, file, o.fs.Lookup(name).DefValue)
	}
	*o.database = pick("database", envDatabase, cfg.Database)
	*o.secret = pick("secret-key", envSecretKey, "")
	*o.sidecar = pick("sidecar", envSidecar, "")
	*o.trusted = pick("trusted-proxies", "", strings.Join(cfg.Server.TrustedProxies, ","))
	*o.uploadLimit = pick("upload-limit", envUploadLimit, cfg.Server.UploadLimit)
	*o.storageLimit = pick("storage-limit", envStorageLimit, cfg.Server.StorageLimit)
	*o.tlsMode = pick("tls", envTLS, cfg.TLS.Mode)
	*o.tlsListen = pick("tls-listen", envTLSListen, cfg.TLS.Listen)
	*o.tlsCert = pick("tls-cert", envTLSCert, cfg.TLS.Cert)
	*o.tlsKey = pick("tls-key", envTLSKey, cfg.TLS.Key)
	*o.acmeEmail = pick("acme-email", envACMEEmail, cfg.TLS.ACMEEmail)
	*o.acmeDirectory = pick("acme-directory", envACMEDir, cfg.TLS.ACMEDirectory)

	chosen := func(name, env, file string) bool {
		return given(o.fs, name) || (env != "" && os.Getenv(env) != "") || file != ""
	}
	o.automatic = !chosen("tls", envTLS, cfg.TLS.Mode) && !chosen("listen", "", cfg.Server.Listen) && *o.trusted == ""

	mode, err := entry.ParseMode(*o.tlsMode)
	if err != nil {
		return mode, err
	}
	fallback := defaultListen
	if mode != entry.Off {
		fallback = defaultTLSPlain
	}
	*o.listen = setting(o.fs, "listen", "", cfg.Server.Listen, fallback)
	return mode, nil
}

func (o *serveOptions) resolveDev(cfg config.File) (entry.Mode, error) {
	if given(o.fs, "tls") && *o.tlsMode != string(entry.Off) {
		return "", fmt.Errorf("-dev serves plain HTTP only, so it cannot be combined with -tls=%s", *o.tlsMode)
	}
	*o.tlsMode = string(entry.Off)
	*o.trusted = ""
	if given(o.fs, "listen") {
		local, err := localAddress(*o.listen)
		if err != nil {
			return "", err
		}
		*o.listen = local
	} else {
		*o.listen = defaultListen
	}
	*o.database = setting(o.fs, "database", envDatabase, cfg.Database, "")
	*o.secret = setting(o.fs, "secret-key", envSecretKey, "", "")
	*o.sidecar = setting(o.fs, "sidecar", envSidecar, "", "")
	*o.uploadLimit = setting(o.fs, "upload-limit", envUploadLimit, cfg.Server.UploadLimit, "0")
	*o.storageLimit = setting(o.fs, "storage-limit", envStorageLimit, cfg.Server.StorageLimit, "0")
	return entry.Off, nil
}

func localAddress(given string) (string, error) {
	value := given
	if !strings.Contains(value, ":") {
		value = ":" + value
	}
	host, port, err := net.SplitHostPort(value)
	if n, perr := strconv.Atoi(port); err != nil || perr != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("-listen %q is not a port or an address with a port", given)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if !loopback(host) {
		return "", fmt.Errorf("-dev only listens on this machine, and -listen %q can be reached from others", given)
	}
	return net.JoinHostPort(host, port), nil
}

func loopback(host string) bool {
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}

func (o *serveOptions) addresses(mode entry.Mode) []string {
	if mode == entry.Off {
		return []string{*o.listen}
	}
	return []string{*o.listen, *o.tlsListen}
}

// Install cannot read the database, as the bundled PostgreSQL refuses root, so it plans for HTTPS.
func (o *serveOptions) installAddresses(mode entry.Mode) []string {
	if o.automatic {
		return []string{defaultTLSPlain, *o.tlsListen}
	}
	return o.addresses(mode)
}

func (o *serveOptions) promote(hosts []string) (string, bool) {
	if !o.automatic {
		return "", false
	}
	for _, host := range hosts {
		if site.PublicHost(host) {
			*o.listen = defaultTLSPlain
			return host, true
		}
	}
	return "", false
}

func setting(fs *flag.FlagSet, name, env, file, fallback string) string {
	if given(fs, name) {
		return fs.Lookup(name).Value.String()
	}
	if env != "" {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}
	if file != "" {
		return file
	}
	return fallback
}

func exposedPorts(addresses []string) []int {
	var ports []int
	for _, addr := range addresses {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		if loopback(host) {
			continue
		}
		n, err := strconv.Atoi(port)
		if err != nil || n <= 0 || slices.Contains(ports, n) {
			continue
		}
		ports = append(ports, n)
	}
	return ports
}
