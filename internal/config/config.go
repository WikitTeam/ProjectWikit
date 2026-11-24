// Package config answers what a person wrote into pwikit.toml.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

type File struct {
	Database  string    `toml:"database"`
	Server    Server    `toml:"server"`
	TLS       TLS       `toml:"tls"`
	Mail      Mail      `toml:"mail"`
	Analytics Analytics `toml:"analytics"`
	Update    Update    `toml:"update"`
	Wikidot   Wikidot   `toml:"wikidot"`
	Mounts    []Mount   `toml:"mount"`
	Storage   Storage   `toml:"storage"`
	Uploads   Uploads   `toml:"uploads"`
}

type StorageBackend struct {
	Mode        string `toml:"mode"`
	Endpoint    string `toml:"endpoint"`
	Region      string `toml:"region"`
	Bucket      string `toml:"bucket"`
	Prefix      string `toml:"prefix"`
	PathStyle   bool   `toml:"path_style"`
	PublicURL   string `toml:"public_url"`
	Credentials string `toml:"credentials"`
}

type Storage struct {
	StorageBackend
	Sites map[string]StorageBackend `toml:"sites"`
}

type UploadLimits struct {
	MaxFile   string `toml:"max_file"`
	SiteQuota string `toml:"site_quota"`
}

type Uploads struct {
	UploadLimits
	KeepDeleted bool                    `toml:"keep_deleted"`
	Sites       map[string]UploadLimits `toml:"sites"`
}

type Mount struct {
	Path        string `toml:"path"`
	Upstream    string `toml:"upstream"`
	StripPrefix bool   `toml:"strip_prefix"`
	SecretFile  string `toml:"secret_file"`
}

type Update struct {
	Auto         *bool  `toml:"auto"`
	PublicBanner *bool  `toml:"public_banner"`
	Check        *bool  `toml:"check"`
	Window       string `toml:"window"`
	TimeZone     string `toml:"time_zone"`
	MinAge       string `toml:"min_age"`
	Mirror       string `toml:"mirror"`
}

type Server struct {
	Listen         string   `toml:"listen"`
	TrustedProxies []string `toml:"trusted_proxies"`
	UploadLimit    string   `toml:"upload_limit"`
	StorageLimit   string   `toml:"storage_limit"`
}

type TLS struct {
	Mode          string `toml:"mode"`
	Listen        string `toml:"listen"`
	Cert          string `toml:"cert"`
	Key           string `toml:"key"`
	ACMEEmail     string `toml:"acme_email"`
	ACMEDirectory string `toml:"acme_directory"`
}

type Mail struct {
	Engine      string `toml:"engine"`
	Host        string `toml:"host"`
	Port        int    `toml:"port"`
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	UseTLS      *bool  `toml:"use_tls"`
	ImplicitTLS *bool  `toml:"implicit_tls"`
	From        string `toml:"from"`
}

type Wikidot struct {
	Username       string `toml:"username"`
	Password       string `toml:"password"`
	MessageSubject string `toml:"message_subject"`
	MessageBody    string `toml:"message_body"`
}

type Analytics struct {
	GoogleTagID string `toml:"google_tag_id"`
}

const (
	EngineSMTP    = "smtp"
	EngineConsole = "console"
)

const FDEnv = "PWIKIT_CONFIG_FD"

var piped struct {
	once sync.Once
	data []byte
	err  error
}

func Load(path string) (File, error) {
	var f File
	var meta toml.MetaData
	var err error
	if raw := os.Getenv(FDEnv); raw != "" || piped.data != nil {
		piped.once.Do(func() { piped.data, piped.err = readInherited(raw) })
		if piped.err != nil {
			return File{}, fmt.Errorf("read %s from descriptor %s: %w", path, raw, piped.err)
		}
		meta, err = toml.Decode(string(piped.data), &f)
	} else {
		meta, err = toml.DecodeFile(path, &f)
		if errors.Is(err, fs.ErrNotExist) {
			return File{}, nil
		}
	}
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	return checked(f, meta, path)
}

func Parse(data []byte, path string) (File, error) {
	var f File
	meta, err := toml.Decode(string(data), &f)
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	return checked(f, meta, path)
}

func checked(f File, meta toml.MetaData, path string) (File, error) {
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		keys := make([]string, len(unknown))
		for i, key := range unknown {
			keys[i] = key.String()
		}
		sort.Strings(keys)
		return File{}, fmt.Errorf("%s has settings pwikit does not know: %s", path, strings.Join(keys, ", "))
	}
	for _, m := range f.Mounts {
		if m.Path == "" || m.Upstream == "" || m.SecretFile == "" {
			return File{}, fmt.Errorf("%s has a [[mount]] without path, upstream or secret_file", path)
		}
		if strings.ContainsAny(m.SecretFile, `/\`) {
			return File{}, fmt.Errorf("%s sets mount secret_file %q; it takes a file name in the secrets folder", path, m.SecretFile)
		}
	}
	for _, name := range append([]string{""}, sortedKeys(f.Storage.Sites)...) {
		b := f.Storage.StorageBackend
		where := "[storage]"
		if name != "" {
			b, where = f.Storage.Site(name), "[storage.sites."+name+"]"
		}
		if err := b.check(); err != nil {
			return File{}, fmt.Errorf("%s %s %w", path, where, err)
		}
	}
	if e := f.Mail.Engine; e != "" && e != EngineSMTP && e != EngineConsole {
		return File{}, fmt.Errorf("%s sets mail.engine to %q; it takes %q or %q", path, e, EngineSMTP, EngineConsole)
	}
	return f, nil
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// A site's backend names only what differs, so the endpoint, region and keys
// carry over from [storage] unless the site sets its own.
func (s Storage) Site(name string) StorageBackend {
	b, ok := s.Sites[name]
	if !ok {
		return s.StorageBackend
	}
	if b.Mode == "" {
		b.Mode = s.Mode
	}
	if b.Endpoint == "" {
		b.Endpoint, b.PathStyle = s.Endpoint, s.PathStyle
	}
	if b.Region == "" {
		b.Region = s.Region
	}
	if b.Bucket == "" {
		b.Bucket = s.Bucket
	}
	if b.Credentials == "" {
		b.Credentials = s.Credentials
	}
	if b.PublicURL == "" && b.Bucket == s.Bucket && b.Prefix == s.Prefix {
		b.PublicURL = s.PublicURL
	}
	return b
}

func (b StorageBackend) check() error {
	switch strings.ToLower(b.Mode) {
	case "", "local":
		return nil
	case "proxy", "direct":
	default:
		return fmt.Errorf("sets mode to %q; it takes local, proxy or direct", b.Mode)
	}
	if b.Endpoint == "" || b.Bucket == "" {
		return errors.New("needs endpoint and bucket when mode is proxy or direct")
	}
	if strings.EqualFold(b.Mode, "direct") && b.PublicURL == "" {
		return errors.New("needs public_url when mode is direct")
	}
	if strings.ContainsAny(b.Credentials, `/\.`) {
		return fmt.Errorf("sets credentials to %q; it takes a plain name", b.Credentials)
	}
	return nil
}

func WriteTemplate(path string) (bool, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := f.WriteString(Template); err != nil {
		f.Close()
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, f.Close()
}

func ReadableByOthers(path string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm()&0o077 != 0
}

const Template = `# Settings for pwikit. A line starting with # is an example and changes nothing
# until the # is taken away.
# A flag on the command line wins over this file, and so does an environment variable.
# Restart pwikit after changing anything here.

# A PostgreSQL of your own. Left unset, pwikit runs the one it carries.
# database = "postgres://user:password@127.0.0.1:5432/pwikit"

[server]
# listen = "127.0.0.1:8080"
# trusted_proxies = ["127.0.0.1"]
# upload_limit = "4GB"
# storage_limit = "0"

[uploads]
# max_file = "35MB"
# site_quota = "0"
# keep_deleted = false

# One site can have its own limits.
# [uploads.sites.main]
# max_file = "100MB"
# site_quota = "10GB"

[storage]
# Left unset, uploads stay in the files folder. proxy keeps them in an S3
# bucket and still serves them through pwikit; direct links readers to
# public_url. The keys go in the secrets folder, see the documentation.
# mode = "proxy"
# endpoint = "https://s3.example.com"
# region = "auto"
# bucket = "pwikit"
# public_url = "https://files.example.org"

# One site can keep its files somewhere else.
# [storage.sites.main]
# mode = "direct"
# bucket = "main-files"
# public_url = "https://main-files.example.org"
# credentials = "main"

[tls]
# Left unset, pwikit serves HTTPS on ports 80 and 443 with certificates from
# Let's Encrypt once a site is bound to a public domain, and plain HTTP on
# 127.0.0.1:8080 until then. Setting listen or trusted_proxies means a proxy sits
# in front, and turns that off.
# off serves plain HTTP, file uses the certificate below, auto always obtains one.
# mode = "auto"
# listen = ":443"
# cert = "/path/to/fullchain.pem"
# key = "/path/to/privkey.pem"
# acme_email = "you@example.com"
# acme_directory = ""

[mail]
# smtp sends mail, console writes it into the log instead.
# engine = "smtp"
# host = "smtp.example.com"
# port = 587
# username = "wiki@example.com"
# password = ""
# use_tls = true
# implicit_tls = false
# from = "wiki@example.com"

[analytics]
# google_tag_id = ""

[update]
# pwikit installed as a system service looks for a new release every hour and
# installs it by itself inside the window below.
# auto = true
# Whether every visitor sees the banner announcing an automatic update, or only
# the people who can open the admin panel.
# public_banner = true
# false stops pwikit from asking for new releases at all.
# check = true
# The hours in which updates are installed, read in time_zone.
# window = "03:00-05:00"
# The time zone the window is read in, such as "Asia/Shanghai". Empty uses this
# machine's time zone.
# time_zone = ""
# How long a release must have been out before it is installed automatically.
# 0s installs it in the next window.
# min_age = "0s"
# A mirror to download from when GitHub cannot be reached. Use only a mirror you trust.
# pwikit update mirror <address> sets this line.
# mirror = ""

[wikidot]
# The Wikidot account pwikit signs in as to send the code that proves someone
# owns an imported Wikidot account. Left unset, a shared service sends it.
# The password can also go in the file wikidot-password in the secrets folder.
# username = ""
# password = ""
# The message that carries the code. {site}, {code} and {minutes} are filled in,
# and the body has to contain {code}. Left empty, the built-in text is sent.
# message_subject = ""
# message_body = """
# """

# Another program on this machine that answers a path of its own. pwikit forwards
# the requests and signs who the reader is in the X-Pwikit-User header, with the
# key in the secrets folder under secret_file. Add one [[mount]] per program.
# [[mount]]
# path = "/app/"
# upstream = "http://127.0.0.1:8100"
# strip_prefix = false
# secret_file = "mount-app.key"
`

func readInherited(raw string) ([]byte, error) {
	os.Unsetenv(FDEnv)
	fd, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "pwikit.toml")
	if f == nil {
		return nil, errors.New("not an open descriptor")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if data == nil {
		data = []byte{}
	}
	return data, err
}
