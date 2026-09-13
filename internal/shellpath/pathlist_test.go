package shellpath

import "testing"

func TestPathWith(t *testing.T) {
	cases := []struct {
		name  string
		value string
		dir   string
		want  string
	}{
		{"empty", "", `C:\pwikit`, `C:\pwikit`},
		{"appended", `C:\Windows;C:\Tools`, `C:\pwikit`, `C:\Windows;C:\Tools;C:\pwikit`},
		{"already there", `C:\Windows;C:\pwikit`, `C:\pwikit`, `C:\Windows;C:\pwikit`},
		{"already there in another case", `C:\Windows;c:\PWIKIT\`, `C:\pwikit`, `C:\Windows;c:\PWIKIT\`},
		{"empty entries dropped", `C:\Windows;;C:\Tools;`, `C:\pwikit`, `C:\Windows;C:\Tools;C:\pwikit`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathWith(tt.value, tt.dir); got != tt.want {
				t.Errorf("pathWith(%q, %q) = %q, want %q", tt.value, tt.dir, got, tt.want)
			}
		})
	}
}

func TestPathWithout(t *testing.T) {
	cases := []struct {
		name  string
		value string
		dir   string
		want  string
	}{
		{"removed", `C:\Windows;C:\old\pwikit;C:\Tools`, `C:\old\pwikit`, `C:\Windows;C:\Tools`},
		{"removed ignoring case and slash", `C:\Windows;c:\OLD\pwikit\`, `C:\old\pwikit`, `C:\Windows`},
		{"not there", `C:\Windows;C:\Tools`, `C:\old\pwikit`, `C:\Windows;C:\Tools`},
		{"only entry", `C:\old\pwikit`, `C:\old\pwikit`, ``},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathWithout(tt.value, tt.dir); got != tt.want {
				t.Errorf("pathWithout(%q, %q) = %q, want %q", tt.value, tt.dir, got, tt.want)
			}
		})
	}
}

func TestPathHasKeepsUnrelatedEntriesApart(t *testing.T) {
	if pathHas(`C:\pwikit-old;C:\Tools`, `C:\pwikit`) {
		t.Errorf("pathHas(%q, %q) = true, want false", `C:\pwikit-old;C:\Tools`, `C:\pwikit`)
	}
}
