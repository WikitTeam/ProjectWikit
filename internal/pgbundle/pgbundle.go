// Package pgbundle answers how pwikit runs the Postgres it ships with.
package pgbundle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/secretfile"
)

const (
	passwordFile = "postgres-password"
	overrideFile = "pwikit.conf"
	versionFile  = "PG_VERSION"

	defaultPort = 5432

	socketPathLimit = 100

	longestSocket = ".s.PGSQL.65535"

	windowsRole = "pwikit"
)

var binaries = []string{"initdb", "pg_ctl", "postgres"}

type Layout struct {
	Root    string
	Bin     string
	Data    string
	Secrets string
}

func NewLayout(root, postgres, data, secrets string) Layout {
	return Layout{Root: root, Bin: filepath.Join(postgres, "bin"), Data: data, Secrets: secrets}
}

func (l Layout) Binary(goos, name string) string {
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join(l.Bin, name)
}

func (l Layout) Locate(goos string) error {
	var missing []string
	for _, name := range binaries {
		if _, err := os.Stat(l.Binary(goos, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("the bundled PostgreSQL is incomplete; %s %s missing from %s.\n"+
		"  Unpack the pwikit release again, or point pwikit at a PostgreSQL you run\n"+
		"  yourself with -database or DATABASE_URL.",
		strings.Join(missing, ", "), plural(len(missing)), l.Bin)
}

func DataVersion(data string) (int, error) {
	raw, err := os.ReadFile(filepath.Join(data, versionFile))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", versionFile, err)
	}
	major, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return 0, fmt.Errorf("%s holds %q, which is not a version", versionFile, strings.TrimSpace(string(raw)))
	}
	return major, nil
}

var versionLine = regexp.MustCompile(`\(PostgreSQL\)\s+(\d+)`)

func ParseVersion(out string) (int, error) {
	found := versionLine.FindStringSubmatch(out)
	if found == nil {
		return 0, fmt.Errorf("cannot read a version out of %q", strings.TrimSpace(out))
	}
	return strconv.Atoi(found[1])
}

func Compatible(dataMajor, binaryMajor int) error {
	if dataMajor == 0 || dataMajor == binaryMajor {
		return nil
	}
	return fmt.Errorf("the data in pgdata/ was written by PostgreSQL %d, and this pwikit carries PostgreSQL %d.\n"+
		"  Nothing was started and nothing was changed. To move the data across:\n"+
		"    1. With the pwikit that made it, run: pwikit backup create\n"+
		"    2. Move pgdata/ aside, then start this pwikit so it makes a fresh one\n"+
		"    3. Stop it, then run: pwikit backup restore <file>",
		dataMajor, binaryMajor)
}

type Plan struct {
	Socket   string
	Host     string
	Port     int
	User     string
	Password string
	Auth     string
	Logs     string
}

func PlanFor(goos string, l Layout, port int, user, password string) Plan {
	if goos == "windows" {
		return Plan{Host: "127.0.0.1", Port: port, User: user, Password: password, Auth: "scram-sha-256"}
	}
	return Plan{Socket: SocketDir(l), Port: port, User: user, Auth: "peer"}
}

func SocketDir(l Layout) string {
	name := filepath.Join(l.Data, longestSocket)
	if len(name) <= socketPathLimit {
		return l.Data
	}
	sum := sha256.Sum256([]byte(l.Root))
	return filepath.Join(os.TempDir(), "pwikit-"+hex.EncodeToString(sum[:4]))
}

func (p Plan) HBA() string {
	if p.Socket != "" {
		return "local all all peer\n"
	}
	return "host all all 127.0.0.1/32 scram-sha-256\n"
}

// An empty listen_addresses turns TCP off entirely.
func (p Plan) Conf() string {
	var b strings.Builder
	b.WriteString("# Written by pwikit on every start. Put your own settings in postgresql.conf.\n")
	fmt.Fprintf(&b, "port = %d\n", p.Port)
	if p.Socket != "" {
		b.WriteString("listen_addresses = ''\n")
		fmt.Fprintf(&b, "unix_socket_directories = '%s'\n", escapeConf(p.Socket))
	} else {
		fmt.Fprintf(&b, "listen_addresses = '%s'\n", p.Host)
		b.WriteString("unix_socket_directories = ''\n")
		b.WriteString("password_encryption = 'scram-sha-256'\n")
	}
	if p.Logs != "" {
		b.WriteString("logging_collector = on\n")
		fmt.Fprintf(&b, "log_directory = '%s'\n", escapeConf(filepath.ToSlash(p.Logs)))
		b.WriteString("log_filename = 'postgresql-%a.log'\n")
		b.WriteString("log_truncate_on_rotation = on\n")
		b.WriteString("log_rotation_age = '1d'\n")
		b.WriteString("log_rotation_size = 0\n")
	}
	return b.String()
}

func (p Plan) DSN(database string) string {
	u := url.URL{Scheme: "postgres", Path: "/" + database}
	if p.Socket != "" {
		u.User = url.User(p.User)
		u.RawQuery = url.Values{"host": {p.Socket}, "port": {strconv.Itoa(p.Port)}}.Encode()
		return u.String()
	}
	u.User = url.UserPassword(p.User, p.Password)
	u.Host = net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	return u.String()
}

// Peer authentication only admits a role named after the OS account.
func RoleFor(goos, account string) string {
	if goos == "windows" {
		return windowsRole
	}
	return account
}

func escapeConf(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func FreePort() (int, error) {
	for range 16 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, fmt.Errorf("find a free port for PostgreSQL: %w", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		if port != defaultPort {
			return port, nil
		}
	}
	return 0, errors.New("find a free port for PostgreSQL: every port offered was 5432")
}

func EnsurePassword(secrets string) (string, error) {
	return secretfile.Ensure(secrets, passwordFile)
}

func ReadPort(data string) (int, error) {
	raw, err := os.ReadFile(filepath.Join(data, overrideFile))
	if err != nil {
		return 0, fmt.Errorf("read the port pwikit chose: %w", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "port" {
			continue
		}
		return strconv.Atoi(strings.TrimSpace(value))
	}
	return 0, fmt.Errorf("%s names no port", overrideFile)
}

// Never connect. Whatever answers on 5432 was not set up for pwikit.
func DefaultPortTaken(timeout time.Duration) bool {
	return taken(net.JoinHostPort("127.0.0.1", strconv.Itoa(defaultPort)), timeout)
}

func taken(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func Hint() string {
	return "Something is listening on port 5432, most likely a PostgreSQL you installed yourself.\n" +
		"  pwikit is starting its own PostgreSQL and leaving that one alone. To use it instead,\n" +
		"  pass -database postgres://user:password@127.0.0.1:5432/dbname or set DATABASE_URL."
}

func plural(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

func WriteConf(data string, p Plan) error {
	name := filepath.Join(data, overrideFile)
	if err := os.WriteFile(name, []byte(p.Conf()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return WireConf(data)
}

func WireConf(data string) error {
	name := filepath.Join(data, "postgresql.conf")
	raw, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	line := fmt.Sprintf("include_if_exists = '%s'", overrideFile)
	if strings.Contains(string(raw), line) {
		return nil
	}
	body := string(raw)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(name, []byte(body+line+"\n"), 0o600)
}

func WriteHBA(data string, p Plan) error {
	name := filepath.Join(data, "pg_hba.conf")
	if err := os.WriteFile(name, []byte(p.HBA()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
