package admin

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

func grantedWith(names ...string) perms.Set {
	return perms.Resolve(perms.Subject{Active: true, Roles: []perms.Role{{ID: 1, Permissions: names}}}, nil)
}

func TestSanctionRowsShowOnlyWhatTheOperatorMaySet(t *testing.T) {
	stored := []db.MemberSanction{{Kind: "ban", Reason: "spam", SetAt: time.Now()}}

	got := sanctionRows(stored, grantedWith(perms.BanMembers, perms.MuteMembers))
	if len(got) != 2 {
		t.Fatalf("len(sanctionRows(ban, mute)) = %d, want 2", len(got))
	}
	if got[0].Kind != "ban" || !got[0].InForce || got[0].Reason != "spam" {
		t.Errorf("sanctionRows()[0] = %+v, want the stored ban", got[0])
	}
	if got[1].Kind != "mute" || got[1].InForce {
		t.Errorf("sanctionRows()[1] = %+v, want mute not in force", got[1])
	}
	if none := sanctionRows(stored, grantedWith(perms.ManageUsers)); len(none) != 0 {
		t.Errorf("sanctionRows(manage users only) = %+v, want none", none)
	}
}

func TestNeededForAction(t *testing.T) {
	cases := map[string]string{
		actionNew:    perms.InviteMembers,
		actionMail:   perms.InviteMembers,
		actionInvite: perms.InviteMembers,
		actionClaim:  perms.InviteMembers,
		actionBot:    perms.ManageBots,
	}
	for action, want := range cases {
		if got := neededForAction(action); got != want {
			t.Errorf("neededForAction(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestEverySanctionKindIsKnownToPerms(t *testing.T) {
	for _, kind := range sanctionKinds {
		if !perms.ValidSanction(kind.Kind) {
			t.Errorf("ValidSanction(%q) = false, want true", kind.Kind)
		}
	}
}
