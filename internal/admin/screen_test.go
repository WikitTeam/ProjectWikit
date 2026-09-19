package admin

import (
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

func TestSuperScreenOpensOnlyForSuperusers(t *testing.T) {
	s := screen{slug: "probe", super: true}
	superuser := perms.Resolve(perms.Subject{Active: true, Superuser: true}, nil)
	sensitive := perms.Resolve(perms.Subject{Active: true, Roles: []perms.Role{{ID: 1, Permissions: []string{perms.ViewSensitiveInfo}}}}, nil)

	if !s.opensFor(superuser) {
		t.Errorf("opensFor(superuser) = false, want true")
	}
	if s.opensFor(sensitive) {
		t.Errorf("opensFor(view_sensitive_info) = true, want false")
	}
}

func TestScreenOpensForItsPermission(t *testing.T) {
	s := screen{slug: "probe", need: perms.ManageTags}
	held := perms.Resolve(perms.Subject{Active: true, Roles: []perms.Role{{ID: 1, Permissions: []string{perms.ManageTags}}}}, nil)

	if !s.opensFor(held) {
		t.Errorf("opensFor(manage_tags) = false, want true")
	}
	if s.opensFor(perms.Resolve(perms.Subject{Active: true}, nil)) {
		t.Errorf("opensFor(nothing) = true, want false")
	}
}

func TestSuspiciousIsSuperuserOnly(t *testing.T) {
	for _, s := range screens {
		if s.slug == suspiciousSlug && !s.super {
			t.Errorf("screen %q super = false, want true", s.slug)
		}
	}
}
