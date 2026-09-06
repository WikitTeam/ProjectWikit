package entry

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseMode(t *testing.T) {
	for raw, want := range map[string]Mode{"off": Off, "file": File, "auto": Auto, " AUTO ": Auto} {
		got, err := ParseMode(raw)
		if err != nil {
			t.Errorf("ParseMode(%q) = _, %v, want %q, nil", raw, err, want)
			continue
		}
		if got != want {
			t.Errorf("ParseMode(%q) = %q, want %q", raw, got, want)
		}
	}
	for _, raw := range []string{"", "on", "acme", "tls"} {
		if got, err := ParseMode(raw); err == nil {
			t.Errorf("ParseMode(%q) = %q, nil, want an error", raw, got)
		}
	}
}

func TestConfigCheckRejectsIncompleteModes(t *testing.T) {
	handler := http.NotFoundHandler()
	cases := map[string]Config{
		"no handler":       {Mode: Off, Plain: ":80"},
		"no plain address": {Mode: Off, Handler: handler},
		"file without pair": {Mode: File, Plain: ":80", Secure: ":443", Handler: handler,
			CertFile: "cert.pem"},
		"file without secure address": {Mode: File, Plain: ":80", Handler: handler,
			CertFile: "cert.pem", KeyFile: "key.pem"},
		"auto without cache": {Mode: Auto, Plain: ":80", Secure: ":443", Handler: handler,
			Hosts: func(context.Context, string) error { return nil }},
		"auto without hosts": {Mode: Auto, Plain: ":80", Secure: ":443", Handler: handler,
			CacheDir: "certs"},
		"unknown mode": {Mode: Mode("on"), Plain: ":80", Handler: handler},
	}
	for name, cfg := range cases {
		if err := cfg.check(); err == nil {
			t.Errorf("check() with %s = nil, want an error", name)
		}
	}
	ok := Config{Mode: Off, Plain: ":80", Handler: handler}
	if err := ok.check(); err != nil {
		t.Errorf("check() with a complete off config = %v, want nil", err)
	}
}

func TestRedirectSecure(t *testing.T) {
	cases := []struct {
		secure string
		host   string
		target string
		want   string
	}{
		{":443", "example.com", "/a/b?c=d", "https://example.com/a/b?c=d"},
		{":443", "example.com:80", "/", "https://example.com/"},
		{"127.0.0.1:8443", "example.com:8080", "/x", "https://example.com:8443/x"},
	}
	for _, c := range cases {
		w := record(t, redirectSecure(c.secure), request(t, http.MethodGet, c.host, c.target))
		if got := w.Header.Get("Location"); got != c.want {
			t.Errorf("redirectSecure(%q) Location for %q = %q, want %q", c.secure, c.host, got, c.want)
		}
		if w.StatusCode != http.StatusFound {
			t.Errorf("redirectSecure(%q) status = %d, want %d", c.secure, w.StatusCode, http.StatusFound)
		}
	}

	w := record(t, redirectSecure(":443"), request(t, http.MethodPost, "example.com", "/"))
	if w.StatusCode != http.StatusBadRequest {
		t.Errorf("redirectSecure POST status = %d, want %d", w.StatusCode, http.StatusBadRequest)
	}
}

func TestSocketNames(t *testing.T) {
	cases := []struct {
		count int
		raw   string
		want  []string
	}{
		{1, "", []string{"http"}},
		{2, "", []string{"http", "https"}},
		{2, "https:http", []string{"https", "http"}},
		{2, "unknown:https", []string{"http", "https"}},
		{3, "", []string{"http", "https", ""}},
	}
	for _, c := range cases {
		got := socketNames(c.count, c.raw)
		if !slices.Equal(got, c.want) {
			t.Errorf("socketNames(%d, %q) = %v, want %v", c.count, c.raw, got, c.want)
		}
	}
}

func TestPrivileged(t *testing.T) {
	for addr, want := range map[string]bool{
		":80": true, ":443": true, "127.0.0.1:8080": false, ":1024": false, ":0": false, "nonsense": false,
	} {
		if got := privileged(addr); got != want {
			t.Errorf("privileged(%q) = %t, want %t", addr, got, want)
		}
	}
}

func TestCertKeeperReloadsAChangedPair(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	first := writePair(t, certFile, keyFile, 1)

	keeper := &certKeeper{certFile: certFile, keyFile: keyFile, log: slog.Default()}
	got, err := keeper.get(nil)
	if err != nil {
		t.Fatalf("get() = _, %v, want a certificate", err)
	}
	if got.Leaf.SerialNumber.Int64() != first {
		t.Errorf("get() serial = %d, want %d", got.Leaf.SerialNumber.Int64(), first)
	}

	second := writePair(t, certFile, keyFile, 2)
	bump(t, certFile, keyFile)
	got, err = keeper.get(nil)
	if err != nil {
		t.Fatalf("get() after renewal = _, %v, want a certificate", err)
	}
	if got.Leaf.SerialNumber.Int64() != second {
		t.Errorf("get() serial after renewal = %d, want %d", got.Leaf.SerialNumber.Int64(), second)
	}
}

func TestCertKeeperKeepsThePairWhenReloadFails(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	serial := writePair(t, certFile, keyFile, 7)

	keeper := &certKeeper{certFile: certFile, keyFile: keyFile,
		log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if _, err := keeper.get(nil); err != nil {
		t.Fatalf("get() = _, %v, want a certificate", err)
	}

	if err := os.WriteFile(certFile, []byte("half written\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bump(t, certFile, keyFile)
	got, err := keeper.get(nil)
	if err != nil {
		t.Fatalf("get() over a broken pair = _, %v, want the previous certificate", err)
	}
	if got.Leaf.SerialNumber.Int64() != serial {
		t.Errorf("get() serial over a broken pair = %d, want %d", got.Leaf.SerialNumber.Int64(), serial)
	}
}

func TestCertKeeperFailsWhenTheFirstLoadFails(t *testing.T) {
	dir := t.TempDir()
	keeper := &certKeeper{certFile: filepath.Join(dir, "missing.pem"),
		keyFile: filepath.Join(dir, "missing.key"), log: slog.Default()}
	if _, err := keeper.reload(); err == nil {
		t.Error("reload() over a missing pair = _, nil, want an error")
	}
}

func TestServeFileModeAnswersBothListeners(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	writePair(t, certFile, keyFile, 3)

	plain, secure := freeAddr(t), freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, Config{
			Mode:     File,
			Plain:    plain,
			Secure:   secure,
			CertFile: certFile,
			KeyFile:  keyFile,
			Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, "served")
			}),
		})
	}()
	waitFor(t, secure)

	pool := x509.NewCertPool()
	pem, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	pool.AppendCertsFromPEM(pem)
	client := &http.Client{
		Transport:     &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	body, status := fetch(t, client, "https://"+secure+"/")
	if status != http.StatusOK || body != "served" {
		t.Errorf("GET https = %d %q, want 200 \"served\"", status, body)
	}

	_, status = fetch(t, client, "http://"+plain+"/")
	if status != http.StatusFound {
		t.Errorf("GET http = %d, want %d", status, http.StatusFound)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve() = %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Error("Serve() did not return after the context was cancelled")
	}
}

func TestServeReportsABusyAddress(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	err = Serve(context.Background(), Config{
		Mode:    Off,
		Plain:   held.Addr().String(),
		Handler: http.NotFoundHandler(),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err == nil {
		t.Fatal("Serve() over a busy address = nil, want an error")
	}
	if !strings.Contains(err.Error(), "for http") {
		t.Errorf("Serve() error = %q, want it to name the http listener", err)
	}
}

func request(t *testing.T, method, host, target string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(method, "http://"+host+target, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = host
	return r
}

func record(t *testing.T, h http.Handler, r *http.Request) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec.Result()
}

func writePair(t *testing.T, certFile, keyFile string, serial int64) int64 {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: "pwikit test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	der8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	write(t, certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	write(t, keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der8}))
	return serial
}

func write(t *testing.T, name string, body []byte) {
	t.Helper()
	if err := os.WriteFile(name, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func bump(t *testing.T, names ...string) {
	t.Helper()
	later := time.Now().Add(time.Minute)
	for _, name := range names {
		if err := os.Chtimes(name, later, later); err != nil {
			t.Fatal(err)
		}
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func waitFor(t *testing.T, addr string) {
	t.Helper()
	for range 200 {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("nothing accepted a connection on %s", addr)
}

func fetch(t *testing.T, client *http.Client, target string) (string, int) {
	t.Helper()
	resp, err := client.Get(target)
	if err != nil {
		t.Fatalf("GET %s = %v", target, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body), resp.StatusCode
}
