package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pwikit.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "pwikit.toml"))
	if err != nil {
		t.Fatalf("Load(missing) err = %v, want nil", err)
	}
	if got.Database != "" || got.Server.Listen != "" || got.Mail.UseTLS != nil {
		t.Errorf("Load(missing) = %+v, want the zero File", got)
	}
}

func TestLoadTemplateIsEmpty(t *testing.T) {
	got, err := Load(writeConfig(t, Template))
	if err != nil {
		t.Fatalf("Load(Template) err = %v, want nil", err)
	}
	if got.Database != "" || got.TLS.Mode != "" || got.Mail.Port != 0 || got.Mail.UseTLS != nil {
		t.Errorf("Load(Template) = %+v, want the zero File", got)
	}
}

func TestLoadTemplateWithEveryLineUncommented(t *testing.T) {
	var lines []string
	for _, line := range strings.Split(Template, "\n") {
		if strings.HasPrefix(line, "# ") && strings.Contains(line, " = ") {
			line = strings.TrimPrefix(line, "# ")
		}
		lines = append(lines, line)
	}
	got, err := Load(writeConfig(t, strings.Join(lines, "\n")))
	if err != nil {
		t.Fatalf("Load(uncommented Template) err = %v, want nil", err)
	}
	if got.Server.Listen != "127.0.0.1:8080" {
		t.Errorf("Server.Listen = %q, want %q", got.Server.Listen, "127.0.0.1:8080")
	}
	if len(got.Server.TrustedProxies) != 1 || got.Server.TrustedProxies[0] != "127.0.0.1" {
		t.Errorf("Server.TrustedProxies = %q, want [127.0.0.1]", got.Server.TrustedProxies)
	}
	if got.Mail.Port != 587 {
		t.Errorf("Mail.Port = %d, want 587", got.Mail.Port)
	}
	if got.Mail.UseTLS == nil || !*got.Mail.UseTLS {
		t.Errorf("Mail.UseTLS = %v, want true", got.Mail.UseTLS)
	}
	if got.Mail.ImplicitTLS == nil || *got.Mail.ImplicitTLS {
		t.Errorf("Mail.ImplicitTLS = %v, want false", got.Mail.ImplicitTLS)
	}
	if got.TLS.ACMEEmail != "you@example.com" {
		t.Errorf("TLS.ACMEEmail = %q, want %q", got.TLS.ACMEEmail, "you@example.com")
	}
}

func TestLoadRefusesAMisspeltKey(t *testing.T) {
	_, err := Load(writeConfig(t, "[tls]\nmod = \"auto\"\n[mail]\nhots = \"x\"\n"))
	if err == nil {
		t.Fatal("Load(misspelt keys) err = nil, want an error")
	}
	for _, key := range []string{"tls.mod", "mail.hots"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Load(misspelt keys) err = %q, want it to name %s", err, key)
		}
	}
}

func TestLoadRefusesAnUnknownMailEngine(t *testing.T) {
	if _, err := Load(writeConfig(t, "[mail]\nengine = \"sendmail\"\n")); err == nil {
		t.Error("Load(engine = sendmail) err = nil, want an error")
	}
}

func TestLoadReportsBrokenSyntax(t *testing.T) {
	if _, err := Load(writeConfig(t, "[server\nlisten = \n")); err == nil {
		t.Error("Load(broken toml) err = nil, want an error")
	}
}

func TestWriteTemplateLeavesAnExistingFileAlone(t *testing.T) {
	path := writeConfig(t, "database = \"mine\"\n")
	wrote, err := WriteTemplate(path)
	if err != nil {
		t.Fatalf("WriteTemplate(existing) err = %v, want nil", err)
	}
	if wrote {
		t.Error("WriteTemplate(existing) = true, want false")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "database = \"mine\"\n" {
		t.Errorf("file after WriteTemplate = %q, want it unchanged", raw)
	}
}

func TestWriteTemplateCreatesAPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pwikit.toml")
	wrote, err := WriteTemplate(path)
	if err != nil || !wrote {
		t.Fatalf("WriteTemplate(missing) = %v, %v, want true, nil", wrote, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("template mode = %o, want 600", info.Mode().Perm())
		}
	}
	if ReadableByOthers(path) {
		t.Errorf("ReadableByOthers(template) = true, want false")
	}
}
