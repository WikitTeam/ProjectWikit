package perms

import "testing"

func TestExplainDecisive(t *testing.T) {
	everyone := Role{ID: 1, Permissions: []string{ViewArticles, EditArticles, CreateForumPosts, CommentArticles}}
	member := Role{ID: 2, Permissions: []string{RateArticles}, Restrictions: []string{EditArticles}}
	tests := []struct {
		name    string
		subject Subject
		object  *Object
		perm    string
		allowed bool
		step    Step
		reason  Reason
		also    int
	}{
		{"granted by a role", Subject{Active: true, Roles: []Role{everyone}}, nil, ViewArticles, true, StepRoles, ReasonRoleAllow, 0},
		{"refused by a role", Subject{Active: true, Roles: []Role{everyone, member}}, nil, EditArticles, false, StepRoles, ReasonRoleDeny, 0},
		{"granted by no role", Subject{Active: true, Roles: []Role{everyone}}, nil, DeleteArticles, false, StepRoles, ReasonNoGrant, 0},
		{"refused by an override", Subject{Active: true, Roles: []Role{everyone}}, &Object{Overrides: []Override{{RoleID: 1, Restrictions: []string{ViewArticles}}}}, ViewArticles, false, StepOverrides, ReasonOverrideDeny, 0},
		{"granted by an override", Subject{Active: true, Roles: []Role{everyone, member}}, &Object{Overrides: []Override{{RoleID: 2, Permissions: []string{EditArticles}}}}, EditArticles, true, StepOverrides, ReasonOverrideAllow, 0},
		{"refused by a sanction", Subject{Active: true, Roles: []Role{everyone}, Sanctions: []Sanction{SanctionMute}}, nil, CreateForumPosts, false, StepSanctions, ReasonSanction, 0},
		{"refused by a locked page", Subject{Active: true, Roles: []Role{everyone}}, &Object{Locked: true}, EditArticles, false, StepObject, ReasonPageLocked, 0},
		{"granted by authorship", Subject{Active: true, Roles: []Role{everyone}}, &Object{Author: true}, ManageArticleAuthors, true, StepObject, ReasonPageAuthor, 0},
		{"granted to a superuser", Subject{Active: true, Superuser: true}, nil, DeleteArticles, true, StepSuperuser, ReasonSuperuser, 0},
		{"refused to an inactive account", Subject{Roles: []Role{everyone}}, nil, ViewArticles, false, StepAccount, ReasonInactive, 0},
		{"refused by every later step", Subject{Active: true, Unverified: true, Roles: []Role{everyone}, Sanctions: []Sanction{SanctionEdit}}, &Object{Locked: true}, EditArticles, false, StepObject, ReasonPageLocked, 2},
		{"no grant and a sanction", Subject{Active: true, Roles: []Role{everyone}, Sanctions: []Sanction{SanctionRating}}, nil, RateArticles, false, StepRoles, ReasonNoGrant, 1},
		{"barred from the forum", Subject{Active: true, Roles: []Role{everyone}}, &Object{Kind: KindForumThread}, CreateForumPosts, false, StepForum, ReasonForumBarred, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Explain(tt.subject, tt.object, tt.perm)
			if d.Allowed != tt.allowed {
				t.Errorf("Explain(%q).Allowed = %v, want %v", tt.perm, d.Allowed, tt.allowed)
			}
			main, also := d.Decisive()
			if main.Step != tt.step {
				t.Errorf("Decisive() step = %d, want %d", main.Step, tt.step)
			}
			if len(main.Causes) == 0 || main.Causes[0].Reason != tt.reason {
				t.Errorf("Decisive() causes = %+v, want first reason %q", main.Causes, tt.reason)
			}
			if len(also) != tt.also {
				t.Errorf("Decisive() also = %d steps, want %d", len(also), tt.also)
			}
		})
	}
}

func TestExplainSkipsStepsThatDoNotApply(t *testing.T) {
	d := Explain(Subject{Active: true}, nil, ViewArticles)
	for _, step := range []Step{StepOverrides, StepObject, StepForum} {
		if got := d.Step(step).Effect; got != EffectSkip {
			t.Errorf("Explain(site-wide).Step(%d).Effect = %d, want %d", step, got, EffectSkip)
		}
	}
}
