package perms

import (
	"math/rand"
	"testing"
)

func referenceResolve(s Subject, o *Object) Set {
	if s.Active && s.Superuser {
		return Set{all: true}
	}
	if !s.Active && !s.Anonymous {
		return Set{named: map[string]bool{}}
	}

	granted := make(map[string]bool)
	refused := make(map[string]bool)
	for _, role := range s.Roles {
		allow := make(map[string]bool, len(role.Permissions))
		for _, name := range role.Permissions {
			allow[name] = true
		}
		deny := make(map[string]bool, len(role.Restrictions))
		for _, name := range role.Restrictions {
			delete(allow, name)
			deny[name] = true
		}
		if o != nil {
			refOverride(allow, deny, o.Overrides, role.ID)
		}
		for name := range allow {
			granted[name] = true
		}
		for name := range deny {
			refused[name] = true
		}
	}
	for name := range refused {
		delete(granted, name)
	}
	if o != nil {
		refObject(granted, o, s)
	}
	if s.Unverified {
		for _, name := range unverified {
			delete(granted, name)
		}
	}
	for _, kind := range s.Sanctions {
		for _, name := range sanctioned(kind) {
			delete(granted, name)
		}
	}
	return Set{named: granted}
}

func refOverride(allow, deny map[string]bool, overrides []Override, roleID int64) {
	for _, override := range overrides {
		if override.RoleID != roleID {
			continue
		}
		for _, name := range override.Permissions {
			allow[name] = true
			delete(deny, name)
		}
		for _, name := range override.Restrictions {
			delete(allow, name)
			deny[name] = true
		}
		return
	}
}

func refObject(granted map[string]bool, o *Object, s Subject) {
	switch o.Kind {
	case KindForumSection:
		if o.HiddenForUsers && granted[ViewForumSections] && !granted[ViewHiddenForumSections] {
			delete(granted, ViewForumSections)
		}
		return
	case KindForumThread:
		refThread(granted, o, s)
		return
	case KindForumPost:
		if o.Author && referenceResolve(s, nil).Has(CreateForumPosts) {
			granted[EditForumPosts] = true
		}
		if o.Thread != nil {
			refThread(granted, o.Thread, s)
		}
		return
	}
	if o.Locked {
		if !granted[LockArticles] {
			for _, name := range lockable {
				delete(granted, name)
			}
		}
		return
	}
	if o.Author {
		granted[ManageArticleAuthors] = true
	}
}

func refThread(granted map[string]bool, o *Object, s Subject) {
	if o.Author {
		granted[EditForumThreads] = true
	}
	barred := !s.Anonymous && !s.ForumActive
	locked := o.Locked && !granted[LockForumThreads]
	if !barred && !locked {
		return
	}
	for _, name := range silenced {
		delete(granted, name)
	}
}

var everyName = func() []string {
	var out []string
	for _, g := range Catalog {
		out = append(out, g.Names...)
	}
	return out
}()

func randomNames(r *rand.Rand) []string {
	var out []string
	for _, name := range everyName {
		if r.Intn(4) == 0 {
			out = append(out, name)
		}
	}
	return out
}

func sparseNames(r *rand.Rand) []string {
	var out []string
	for _, name := range everyName {
		if r.Intn(12) == 0 {
			out = append(out, name)
		}
	}
	return out
}

func randomSubject(r *rand.Rand) Subject {
	s := Subject{
		Anonymous:   r.Intn(6) == 0,
		Active:      r.Intn(8) != 0,
		ForumActive: r.Intn(4) != 0,
		Unverified:  r.Intn(4) == 0,
		Superuser:   r.Intn(10) == 0,
	}
	n := int64(1 + r.Intn(4))
	for id := int64(1); id <= n; id++ {
		role := Role{ID: id, Permissions: randomNames(r)}
		if r.Intn(3) == 0 {
			role.Restrictions = sparseNames(r)
		}
		s.Roles = append(s.Roles, role)
	}
	kinds := []Sanction{SanctionBan, SanctionMute, SanctionEdit, SanctionRating}
	for _, k := range kinds {
		if r.Intn(6) == 0 {
			s.Sanctions = append(s.Sanctions, k)
		}
	}
	return s
}

func randomObject(r *rand.Rand, roles int) *Object {
	if r.Intn(5) == 0 {
		return nil
	}
	o := &Object{
		Kind:           Kind(r.Intn(4)),
		Locked:         r.Intn(3) == 0,
		Author:         r.Intn(3) == 0,
		HiddenForUsers: r.Intn(2) == 0,
	}
	for i := 0; i < r.Intn(4); i++ {
		o.Overrides = append(o.Overrides, Override{RoleID: int64(1 + r.Intn(roles+1)), Permissions: randomNames(r), Restrictions: sparseNames(r)})
	}
	if o.Kind == KindForumPost && r.Intn(4) != 0 {
		o.Thread = &Object{Kind: KindForumThread, Locked: r.Intn(3) == 0, Author: r.Intn(3) == 0, Overrides: o.Overrides}
	}
	return o
}

func TestResolveMatchesReference(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		s := randomSubject(r)
		o := randomObject(r, len(s.Roles))
		got, want := Resolve(s, o), referenceResolve(s, o)
		for _, name := range everyName {
			if got.Has(name) != want.Has(name) {
				t.Fatalf("case %d Resolve(...).Has(%q) = %v, want %v", i, name, got.Has(name), want.Has(name))
			}
		}
	}
}

func TestExplainMatchesResolve(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 5000; i++ {
		s := randomSubject(r)
		o := randomObject(r, len(s.Roles))
		set := Resolve(s, o)
		for _, name := range everyName {
			if got := Explain(s, o, name).Allowed; got != set.Has(name) {
				t.Fatalf("case %d Explain(%q).Allowed = %v, want %v", i, name, got, set.Has(name))
			}
		}
	}
}
