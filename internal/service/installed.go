package service

import (
	"regexp"
	"strings"
)

type Installed struct {
	Executable       string
	User             string
	UpdateExecutable string
	DataDir          string
	ServeArgs        []string
}

func serveExtra(args []string) []string {
	i := 0
	for i < len(args) && args[i] != "serve" {
		i++
	}
	if i == len(args) {
		return nil
	}
	var out []string
	for i++; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-data-dir" || arg == "--data-dir" || arg == "-log-file" || arg == "--log-file":
			i++
		case strings.HasPrefix(arg, "-data-dir=") || strings.HasPrefix(arg, "--data-dir=") ||
			strings.HasPrefix(arg, "-log-file=") || strings.HasPrefix(arg, "--log-file="):
		default:
			out = append(out, arg)
		}
	}
	return out
}

func dataDirArg(args []string) string {
	for i, arg := range args {
		if value, ok := strings.CutPrefix(arg, "-data-dir="); ok {
			return value
		}
		if value, ok := strings.CutPrefix(arg, "--data-dir="); ok {
			return value
		}
		if (arg == "-data-dir" || arg == "--data-dir") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func systemdArgs(unit string) []string {
	value, ok := systemdValue(unit, "ExecStart")
	if !ok {
		return nil
	}
	var out []string
	for value != "" {
		var first string
		first, value = systemdFirst(value)
		out = append(out, first)
	}
	return out
}

var plistStrings = regexp.MustCompile(`<string>([^<]*)</string>`)

var plistArray = regexp.MustCompile(`<key>ProgramArguments</key>\s*<array>([\s\S]*?)</array>`)

func plistArgs(text string) []string {
	m := plistArray.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	var out []string
	for _, s := range plistStrings.FindAllStringSubmatch(m[1], -1) {
		out = append(out, xmlUnescape(s[1]))
	}
	return out
}

func systemdValue(unit, key string) (string, bool) {
	for _, line := range strings.Split(unit, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), key+"="); ok {
			return value, true
		}
	}
	return "", false
}

func systemdFirst(value string) (first, rest string) {
	value = strings.TrimLeft(value, " ")
	if !strings.HasPrefix(value, `"`) {
		first, rest, _ = strings.Cut(value, " ")
		return strings.ReplaceAll(first, "%%", "%"), rest
	}
	var b strings.Builder
	for i := 1; i < len(value); i++ {
		c := value[i]
		switch {
		case c == '\\' && i+1 < len(value):
			i++
			b.WriteByte(value[i])
		case (c == '$' || c == '%') && i+1 < len(value) && value[i+1] == c:
			i++
			b.WriteByte(c)
		case c == '"':
			return b.String(), strings.TrimLeft(value[i+1:], " ")
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), ""
}

func systemdProgram(unit string) string {
	value, ok := systemdValue(unit, "ExecStart")
	if !ok {
		return ""
	}
	first, _ := systemdFirst(value)
	return first
}

func pointSystemdAt(unit, exe string) (string, bool) {
	lines := strings.Split(unit, "\n")
	for i, line := range lines {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "ExecStart=")
		if !ok {
			continue
		}
		first, rest := systemdFirst(value)
		if first == exe {
			return unit, false
		}
		lines[i] = strings.TrimRight("ExecStart="+systemdQuote(exe)+" "+rest, " ")
		return strings.Join(lines, "\n"), true
	}
	return unit, false
}

var plistProgram = regexp.MustCompile(`(<key>ProgramArguments</key>\s*<array>\s*<string>)([^<]*)(</string>)`)

var plistUser = regexp.MustCompile(`<key>UserName</key>\s*<string>([^<]*)</string>`)

func plistFirst(text string) string {
	m := plistProgram.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return xmlUnescape(m[2])
}

func plistUserName(text string) string {
	m := plistUser.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return xmlUnescape(m[1])
}

func pointPlistAt(text, exe string) (string, bool) {
	m := plistProgram.FindStringSubmatchIndex(text)
	if m == nil || xmlUnescape(text[m[4]:m[5]]) == exe {
		return text, false
	}
	return text[:m[4]] + xmlText(exe) + text[m[5]:], true
}

func xmlUnescape(s string) string {
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#34;", `"`, "&apos;", "'", "&#39;", "'", "&#x9;", "\t", "&#xA;", "\n", "&#xD;", "\r", "&amp;", "&").Replace(s)
}
