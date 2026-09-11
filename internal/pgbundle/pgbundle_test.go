package pgbundle

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func layout(t *testing.T) Layout {
	t.Helper()
	root := t.TempDir()
	return NewLayout(root, filepath.Join(root, "postgres"), filepath.Join(root, "pgdata"), filepath.Join(root, "secrets"))
}

func TestBinaryAddsTheWindowsSuffix(t *testing.T) {
	l := Layout{Bin: "bin"}
	if got := l.Binary("windows", "initdb"); got != filepath.Join("bin", "initdb.exe") {
		t.Errorf("Binary(windows) = %q, want %q", got, filepath.Join("bin", "initdb.exe"))
	}
	if got := l.Binary("linux", "initdb"); got != filepath.Join("bin", "initdb") {
		t.Errorf("Binary(linux) = %q, want %q", got, filepath.Join("bin", "initdb"))
	}
}

func TestLocateNamesEveryMissingProgram(t *testing.T) {
	l := layout(t)
	if err := os.MkdirAll(l.Bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.Binary("linux", "postgres"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	err := l.Locate("linux")
	if err == nil {
		t.Fatal("Locate() err = nil, want non-nil")
	}
	for _, want := range []string{"initdb", "pg_ctl", "-database"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Contains(Locate(), %q) = false, want true", want)
		}
	}
	if strings.Contains(err.Error(), "postgres,") {
		t.Errorf("Locate() = %v, want postgres left out because it is present", err)
	}
}

func TestLocatePassesAWholeBundle(t *testing.T) {
	l := layout(t)
	if err := os.MkdirAll(l.Bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range binaries {
		if err := os.WriteFile(l.Binary("linux", name), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Locate("linux"); err != nil {
		t.Errorf("Locate() err = %v, want nil", err)
	}
}

func TestDataVersionIsZeroBeforeInitdb(t *testing.T) {
	got, err := DataVersion(filepath.Join(t.TempDir(), "never-made"))
	if err != nil {
		t.Fatalf("DataVersion() err = %v, want nil", err)
	}
	if got != 0 {
		t.Errorf("DataVersion() = %d, want 0", got)
	}
}

func TestDataVersionReadsTheFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, versionFile), []byte("17\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := DataVersion(dir)
	if err != nil {
		t.Fatalf("DataVersion() err = %v, want nil", err)
	}
	if got != 17 {
		t.Errorf("DataVersion() = %d, want 17", got)
	}
}

func TestDataVersionRejectsAGarbledFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, versionFile), []byte("seventeen"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DataVersion(dir); err == nil {
		t.Error("DataVersion(garbled) err = nil, want non-nil")
	}
}

func TestParseVersion(t *testing.T) {
	for _, c := range []struct {
		out  string
		want int
	}{
		{"postgres (PostgreSQL) 17.2\n", 17},
		{"postgres (PostgreSQL) 14.23", 14},
		{"postgres (PostgreSQL) 18beta1", 18},
	} {
		got, err := ParseVersion(c.out)
		if err != nil {
			t.Errorf("ParseVersion(%q) err = %v, want nil", c.out, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseVersion(%q) = %d, want %d", c.out, got, c.want)
		}
	}
	if _, err := ParseVersion("command not found"); err == nil {
		t.Error("ParseVersion(junk) err = nil, want non-nil")
	}
}

func TestCompatibleLetsAFreshOrMatchingDirectoryThrough(t *testing.T) {
	for _, c := range [][2]int{{0, 17}, {17, 17}} {
		if err := Compatible(c[0], c[1]); err != nil {
			t.Errorf("Compatible(%d, %d) err = %v, want nil", c[0], c[1], err)
		}
	}
}

func TestCompatibleExplainsAMismatch(t *testing.T) {
	err := Compatible(16, 17)
	if err == nil {
		t.Fatal("Compatible(16, 17) err = nil, want non-nil")
	}
	for _, want := range []string{"16", "17", "pwikit backup create", "pwikit backup restore", "Nothing was started"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Contains(Compatible(16, 17), %q) = false, want true", want)
		}
	}
}

func TestPlanForWindowsTakesAPasswordOverLoopback(t *testing.T) {
	p := PlanFor("windows", layout(t), 54321, "pwikit", "secret")
	if p.Socket != "" {
		t.Errorf("PlanFor(windows).Socket = %q, want none", p.Socket)
	}
	if p.Host != "127.0.0.1" {
		t.Errorf("PlanFor(windows).Host = %q, want 127.0.0.1", p.Host)
	}
	if !strings.Contains(p.HBA(), "scram-sha-256") {
		t.Errorf("HBA() = %q, want scram-sha-256", p.HBA())
	}
	if strings.Contains(p.HBA(), "trust") {
		t.Errorf("HBA() = %q, want no trust line", p.HBA())
	}
}

func TestPlanForUnixUsesThePeerOverASocketAndNoPort(t *testing.T) {
	p := PlanFor("linux", layout(t), 54321, "kakushi", "")
	if p.Socket == "" {
		t.Fatal("PlanFor(linux).Socket = empty, want a directory")
	}
	if p.Password != "" {
		t.Errorf("PlanFor(linux).Password = %q, want none", p.Password)
	}
	if p.HBA() != "local all all peer\n" {
		t.Errorf("HBA() = %q, want only the peer line", p.HBA())
	}
	if !strings.Contains(p.Conf(), "listen_addresses = ''") {
		t.Errorf("Conf() = %q, want TCP switched off", p.Conf())
	}
}

func TestSocketDirStaysInsideAShortStateDirectory(t *testing.T) {
	l := Layout{Root: "/srv/wiki", Data: "/srv/wiki/pgdata"}
	if got := SocketDir(l); got != "/srv/wiki/pgdata" {
		t.Errorf("SocketDir(short) = %q, want the data directory", got)
	}
}

func TestSocketDirMovesOutOfADeepStateDirectory(t *testing.T) {
	deep := "/" + strings.Repeat("deep/", 30)
	l := Layout{Root: deep, Data: deep + "pgdata"}
	got := SocketDir(l)
	if strings.HasPrefix(got, deep) {
		t.Errorf("SocketDir(deep) = %q, want somewhere shorter", got)
	}
	if len(filepath.Join(got, longestSocket)) > socketPathLimit {
		t.Errorf("len(socket) = %d, want at most %d", len(filepath.Join(got, longestSocket)), socketPathLimit)
	}
	if again := SocketDir(l); again != got {
		t.Errorf("SocketDir() = %q then %q, want the same place both times", got, again)
	}
}

func TestDSNSurvivesAnAwkwardAccountName(t *testing.T) {
	p := Plan{Host: "127.0.0.1", Port: 54321, User: `DOMAIN\Some One`, Password: "p@ss/word"}
	cfg, err := pgx.ParseConfig(p.DSN("pwikit"))
	if err != nil {
		t.Fatalf("ParseConfig(DSN()) err = %v, want nil", err)
	}
	if cfg.User != `DOMAIN\Some One` {
		t.Errorf("ParseConfig(DSN()).User = %q, want the account back", cfg.User)
	}
	if cfg.Password != "p@ss/word" {
		t.Errorf("ParseConfig(DSN()).Password = %q, want the password back", cfg.Password)
	}
	if cfg.Port != 54321 {
		t.Errorf("ParseConfig(DSN()).Port = %d, want 54321", cfg.Port)
	}
}

func TestDSNPointsASocketPlanAtTheDirectory(t *testing.T) {
	p := Plan{Socket: "/srv/my wiki/pgdata", Port: 54321, User: "kakushi"}
	u, err := url.Parse(p.DSN("pwikit"))
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("host"); got != "/srv/my wiki/pgdata" {
		t.Errorf("DSN().host = %q, want the socket directory", got)
	}
}

func TestRoleFor(t *testing.T) {
	if got := RoleFor("linux", "kakushi"); got != "kakushi" {
		t.Errorf("RoleFor(linux) = %q, want the account", got)
	}
	if got := RoleFor("windows", `DOMAIN\kakushi`); got != "pwikit" {
		t.Errorf("RoleFor(windows) = %q, want pwikit", got)
	}
}

func TestFreePortIsNeverTheDefault(t *testing.T) {
	for range 20 {
		port, err := FreePort()
		if err != nil {
			t.Fatalf("FreePort() err = %v, want nil", err)
		}
		if port == defaultPort {
			t.Fatalf("FreePort() = %d, want anything but 5432", port)
		}
	}
}

func TestWriteConfThenReadPort(t *testing.T) {
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(data, "postgresql.conf"), []byte("shared_buffers = 128MB"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := PlanFor("windows", Layout{Data: data}, 54999, "pwikit", "x")
	if err := WriteConf(data, p); err != nil {
		t.Fatalf("WriteConf() err = %v, want nil", err)
	}
	got, err := ReadPort(data)
	if err != nil {
		t.Fatalf("ReadPort() err = %v, want nil", err)
	}
	if got != 54999 {
		t.Errorf("ReadPort() = %d, want 54999", got)
	}
}

func TestWireConfAddsTheIncludeOnce(t *testing.T) {
	data := t.TempDir()
	conf := filepath.Join(data, "postgresql.conf")
	if err := os.WriteFile(conf, []byte("shared_buffers = 128MB"), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := WireConf(data); err != nil {
			t.Fatalf("WireConf() err = %v, want nil", err)
		}
	}
	raw, err := os.ReadFile(conf)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), "include_if_exists"); n != 1 {
		t.Errorf("include lines = %d, want 1", n)
	}
	if !strings.HasPrefix(string(raw), "shared_buffers = 128MB\n") {
		t.Errorf("postgresql.conf = %q, want the original settings kept", raw)
	}
}

func TestWriteHBAReplacesTrust(t *testing.T) {
	data := t.TempDir()
	hba := filepath.Join(data, "pg_hba.conf")
	if err := os.WriteFile(hba, []byte("local all all trust\nhost all all 127.0.0.1/32 trust\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteHBA(data, PlanFor("windows", Layout{Data: data}, 54999, "pwikit", "x")); err != nil {
		t.Fatalf("WriteHBA() err = %v, want nil", err)
	}
	raw, err := os.ReadFile(hba)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "trust") {
		t.Errorf("pg_hba.conf = %q, want no trust left", raw)
	}
}

func TestTakenOnlyLooks(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	sent := make(chan bool, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			sent <- false
			return
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, _ := conn.Read(make([]byte, 1))
		sent <- n > 0
	}()

	if !taken(l.Addr().String(), time.Second) {
		t.Error("taken(a listening address) = false, want true")
	}
	if <-sent {
		t.Error("taken() wrote to the listener, want it to connect and leave")
	}
}

func TestTakenIsFalseForASilentAddress(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	if taken(addr, 200*time.Millisecond) {
		t.Errorf("taken(%s) = true after the listener closed, want false", addr)
	}
}

func TestHintSaysHowToUseTheOtherServer(t *testing.T) {
	for _, want := range []string{"5432", "-database", "DATABASE_URL"} {
		if !strings.Contains(Hint(), want) {
			t.Errorf("Contains(Hint(), %q) = false, want true", want)
		}
	}
}
