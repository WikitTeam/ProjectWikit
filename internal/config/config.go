// Package config answers what a person wrote into pwikit.toml.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type File struct {
	Database  string    `toml:"database"`
	Server    Server    `toml:"server"`
	TLS       TLS       `toml:"tls"`
	Mail      Mail      `toml:"mail"`
	Analytics Analytics `toml:"analytics"`
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

type Analytics struct {
	GoogleTagID string `toml:"google_tag_id"`
}

const (
	EngineSMTP    = "smtp"
	EngineConsole = "console"
)

func Load(path string) (File, error) {
	var f File
	meta, err := toml.DecodeFile(path, &f)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		keys := make([]string, len(unknown))
		for i, key := range unknown {
			keys[i] = key.String()
		}
		sort.Strings(keys)
		return File{}, fmt.Errorf("%s has settings pwikit does not know: %s", path, strings.Join(keys, ", "))
	}
	if e := f.Mail.Engine; e != "" && e != EngineSMTP && e != EngineConsole {
		return File{}, fmt.Errorf("%s sets mail.engine to %q; it takes %q or %q", path, e, EngineSMTP, EngineConsole)
	}
	return f, nil
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
`
