package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// The file is edited as text rather than decoded and encoded again, so the
// comments and the order a person gave it survive.
func SetUpdateMirror(path, mirror string) error {
	if err := CheckMirror(mirror); err != nil {
		return err
	}
	if _, err := WriteTemplate(path); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	edited := setMirrorLine(string(raw), mirror)
	if err := os.WriteFile(path, []byte(edited), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := Load(path); err != nil {
		if restoreErr := os.WriteFile(path, raw, info.Mode().Perm()); restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		return err
	}
	return nil
}

func CheckMirror(mirror string) error {
	if mirror == "" {
		return nil
	}
	u, err := url.Parse(mirror)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("mirror %q is not an http or https address", mirror)
	}
	for _, r := range mirror {
		if r < 0x21 || r > 0x7e || r == '"' || r == '\\' {
			return fmt.Errorf("mirror %q holds a character a mirror address cannot have", mirror)
		}
	}
	return nil
}

func setMirrorLine(text, mirror string) string {
	nl := "\n"
	if strings.Contains(text, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	value := `mirror = "` + strings.TrimSuffix(mirror, "/") + `"`
	if mirror == "" {
		value = `# mirror = ""`
	}

	section, header, set, example := "", -1, -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			name, _, _ := strings.Cut(strings.TrimPrefix(trimmed, "["), "]")
			section = strings.TrimSpace(name)
			if section == "update" && header < 0 {
				header = i
			}
			continue
		}
		if section != "update" {
			continue
		}
		key, _, found := strings.Cut(strings.TrimSpace(strings.TrimPrefix(trimmed, "#")), "=")
		if !found || strings.TrimSpace(key) != "mirror" {
			continue
		}
		switch {
		case !strings.HasPrefix(trimmed, "#") && set < 0:
			set = i
		case strings.HasPrefix(trimmed, "#") && example < 0:
			example = i
		}
	}

	switch {
	case set >= 0:
		lines[set] = value
	case example >= 0:
		lines[example] = value
	case header >= 0:
		lines = append(lines[:header+1], append([]string{value}, lines[header+1:]...)...)
	default:
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, "", "[update]", value, "")
	}
	return strings.Join(lines, nl)
}
