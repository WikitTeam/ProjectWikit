// Package entry answers how pwikit gets its listening sockets and who
// terminates TLS.
package entry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type Mode string

const (
	Off  Mode = "off"
	File Mode = "file"
	Auto Mode = "auto"
)

var modes = []Mode{Off, File, Auto}

const (
	socketPlain  = "http"
	socketSecure = "https"

	readHeaderTimeout = 20 * time.Second
	shutdownGrace     = 15 * time.Second
)

func ParseMode(raw string) (Mode, error) {
	m := Mode(strings.ToLower(strings.TrimSpace(raw)))
	if slices.Contains(modes, m) {
		return m, nil
	}
	return "", fmt.Errorf("unknown tls mode %q, want off, file or auto", raw)
}

type Hosts func(ctx context.Context, host string) error

type Config struct {
	Mode      Mode
	Plain     string
	Secure    string
	CertFile  string
	KeyFile   string
	CacheDir  string
	Email     string
	Directory string
	Hosts     Hosts
	Handler   http.Handler
	Logger    *slog.Logger
}

func (c Config) check() error {
	if c.Handler == nil {
		return errors.New("no handler")
	}
	if c.Plain == "" {
		return errors.New("no plain listen address")
	}
	switch c.Mode {
	case Off:
		return nil
	case File:
		if c.CertFile == "" || c.KeyFile == "" {
			return errors.New("tls mode file needs a certificate and a private key")
		}
	case Auto:
		if c.CacheDir == "" {
			return errors.New("tls mode auto needs a directory to keep certificates in")
		}
		if c.Hosts == nil {
			return errors.New("tls mode auto needs to know which hosts belong to this server")
		}
	default:
		return fmt.Errorf("unknown tls mode %q", c.Mode)
	}
	if c.Secure == "" {
		return errors.New("no https listen address")
	}
	return nil
}

func Serve(ctx context.Context, cfg Config) error {
	if err := cfg.check(); err != nil {
		return err
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	plain := cfg.Handler
	var secure *http.Server

	switch cfg.Mode {
	case File:
		keeper := &certKeeper{certFile: cfg.CertFile, keyFile: cfg.KeyFile, recheck: time.Minute, log: log}
		if _, err := keeper.reload(); err != nil {
			return err
		}
		plain = redirectSecure(cfg.Secure)
		secure = server(cfg.Handler, &tls.Config{GetCertificate: keeper.get, MinVersion: tls.VersionTLS12})
	case Auto:
		manager := &autocert.Manager{
			Cache:      autocert.DirCache(cfg.CacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostPolicy(cfg.Hosts),
			Email:      cfg.Email,
		}
		if cfg.Directory != "" {
			manager.Client = &acme.Client{DirectoryURL: cfg.Directory}
		}
		plain = manager.HTTPHandler(nil)
		secure = server(cfg.Handler, manager.TLSConfig())
	}

	in := inheritedSockets()
	plainListener, err := listen(socketPlain, cfg.Plain, in)
	if err != nil {
		return err
	}
	running := []*http.Server{server(plain, nil)}
	listeners := []net.Listener{plainListener}
	if secure != nil {
		secureListener, err := listen(socketSecure, cfg.Secure, in)
		if err != nil {
			plainListener.Close()
			return err
		}
		running = append(running, secure)
		listeners = append(listeners, tls.NewListener(secureListener, secure.TLSConfig))
	}

	log.Info("pwikit listening", "tls", string(cfg.Mode), "plain", plainListener.Addr().String(),
		"secure", secureAddr(listeners), "inherited", len(in))

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errs := make(chan error, len(running))
	for i, s := range running {
		go func() { errs <- s.Serve(listeners[i]) }()
	}

	var first error
	select {
	case first = <-errs:
	case <-ctx.Done():
	}
	stop()

	closing, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace)
	defer cancel()
	var wg sync.WaitGroup
	for _, s := range running {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Shutdown(closing)
		}()
	}
	wg.Wait()

	if first != nil && !errors.Is(first, http.ErrServerClosed) {
		return first
	}
	return nil
}

func server(h http.Handler, conf *tls.Config) *http.Server {
	return &http.Server{Handler: h, TLSConfig: conf, ReadHeaderTimeout: readHeaderTimeout}
}

func secureAddr(listeners []net.Listener) string {
	if len(listeners) < 2 {
		return ""
	}
	return listeners[1].Addr().String()
}

func redirectSecure(secure string) http.Handler {
	_, port, _ := net.SplitHostPort(secure)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		host := site.StripPort(r.Host)
		if port != "" && port != "443" {
			host = net.JoinHostPort(host, port)
		}
		http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusFound)
	})
}

type certKeeper struct {
	certFile string
	keyFile  string
	recheck  time.Duration
	log      *slog.Logger

	mu      sync.Mutex
	cert    *tls.Certificate
	stamp   [2]time.Time
	checked time.Time
}

func (k *certKeeper) get(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.cert != nil && time.Since(k.checked) < k.recheck {
		return k.cert, nil
	}
	return k.reload()
}

func (k *certKeeper) reload() (*tls.Certificate, error) {
	stamp, err := k.stamps()
	k.checked = time.Now()
	if err == nil && k.cert != nil && stamp == k.stamp {
		return k.cert, nil
	}
	pair, err := tls.LoadX509KeyPair(k.certFile, k.keyFile)
	if err != nil {
		if k.cert == nil {
			return nil, fmt.Errorf("load certificate %q with key %q: %w", k.certFile, k.keyFile, err)
		}
		k.log.Error("reload certificate", "cert", k.certFile, "key", k.keyFile, "err", err)
		return k.cert, nil
	}
	k.cert, k.stamp = &pair, stamp
	return k.cert, nil
}

func (k *certKeeper) stamps() ([2]time.Time, error) {
	var stamp [2]time.Time
	for i, name := range []string{k.certFile, k.keyFile} {
		info, err := os.Stat(name)
		if err != nil {
			return stamp, err
		}
		stamp[i] = info.ModTime()
	}
	return stamp, nil
}

func listen(name, addr string, in map[string]*os.File) (net.Listener, error) {
	if f := in[name]; f != nil {
		l, err := net.FileListener(f)
		if err != nil {
			return nil, fmt.Errorf("adopt the %s socket handed over at startup: %w", name, err)
		}
		return l, nil
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		if privileged(addr) && errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("listen on %s for %s: %w. Grant the binary CAP_NET_BIND_SERVICE, hand the socket over with systemd socket activation, or listen above port 1024 behind a proxy", addr, name, err)
		}
		return nil, fmt.Errorf("listen on %s for %s: %w", addr, name, err)
	}
	return l, nil
}

func privileged(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n > 0 && n < 1024
}

func inheritedSockets() map[string]*os.File {
	if strconv.Itoa(os.Getpid()) != os.Getenv("LISTEN_PID") {
		return nil
	}
	count, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || count <= 0 {
		return nil
	}
	out := make(map[string]*os.File, count)
	for i, name := range socketNames(count, os.Getenv("LISTEN_FDNAMES")) {
		if name == "" {
			continue
		}
		out[name] = os.NewFile(uintptr(3+i), name)
	}
	return out
}

func socketNames(count int, raw string) []string {
	given := strings.Split(raw, ":")
	byPosition := []string{socketPlain, socketSecure}
	names := make([]string, count)
	for i := range names {
		if i < len(given) && (given[i] == socketPlain || given[i] == socketSecure) {
			names[i] = given[i]
			continue
		}
		if i < len(byPosition) {
			names[i] = byPosition[i]
		}
	}
	return names
}
