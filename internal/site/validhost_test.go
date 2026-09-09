package site

import "testing"

func TestValidHostAcceptsWhatARequestCarries(t *testing.T) {
	for _, host := range []string{
		"localhost",
		"localhost:8080",
		"example.org",
		"wiki.example.org",
		"media.wiki.example.org",
		"EXAMPLE.org",
		"a-b.example.org",
		"example.org:65535",
	} {
		if !ValidHost(host) {
			t.Errorf("ValidHost(%q) = false, want true", host)
		}
	}
}

func TestValidHostRejectsWhatNoRequestCarries(t *testing.T) {
	for _, host := range []string{
		"",
		"http://example.org",
		"example.org/wiki",
		"example.org.",
		".example.org",
		"exa mple.org",
		"example.org:0",
		"example.org:70000",
		"example.org:http",
		"-example.org",
		"example-.org",
		"exam_ple.org",
	} {
		if ValidHost(host) {
			t.Errorf("ValidHost(%q) = true, want false", host)
		}
	}
}

func TestValidHostRejectsAnOverlongLabel(t *testing.T) {
	label := ""
	for len(label) < 64 {
		label += "a"
	}
	if ValidHost(label + ".example.org") {
		t.Errorf("ValidHost(a 64 character label) = true, want false")
	}
}

func TestDecideIgnoresHostCase(t *testing.T) {
	s := Site{Domain: "wiki.example", MediaDomain: "media.example"}
	got := Decide(s, "MEDIA.EXAMPLE", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v, want Serve", got.Action)
	}
}
