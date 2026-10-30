package perms

import "slices"

type Value int8

const (
	Unset Value = iota
	Allow
	Deny
)

type Step int

const (
	StepAccount Step = iota + 1
	StepSuperuser
	StepRoles
	StepOverrides
	StepObject
	StepForum
	StepEmail
	StepSanctions
)

var Steps = []Step{StepAccount, StepSuperuser, StepRoles, StepOverrides, StepObject, StepForum, StepEmail, StepSanctions}

type Effect int8

const (
	// Skip is for a step that does not apply at all, None for one that ran
	// and changed nothing.
	EffectSkip Effect = iota
	EffectNone
	EffectAllow
	EffectDeny
)

type Reason string

const (
	ReasonSuperuser     Reason = "superuser"
	ReasonInactive      Reason = "inactive"
	ReasonNoGrant       Reason = "no-grant"
	ReasonRoleAllow     Reason = "role-allow"
	ReasonRoleDeny      Reason = "role-deny"
	ReasonOverrideAllow Reason = "override-allow"
	ReasonOverrideDeny  Reason = "override-deny"
	ReasonPageLocked    Reason = "page-locked"
	ReasonPageAuthor    Reason = "page-author"
	ReasonHiddenSection Reason = "hidden-section"
	ReasonThreadAuthor  Reason = "thread-author"
	ReasonThreadLocked  Reason = "thread-locked"
	ReasonPostAuthor    Reason = "post-author"
	ReasonForumBarred   Reason = "forum-barred"
	ReasonUnverified    Reason = "unverified"
	ReasonSanction      Reason = "sanction"
)

type Cause struct {
	Reason   Reason
	Effect   Effect
	RoleID   int64
	Sanction Sanction
}

type StepResult struct {
	Step   Step
	Effect Effect
	Causes []Cause
}

type RoleValue struct {
	RoleID      int64
	Base        Value
	HasOverride bool
	Override    Value
	Result      Value
}

type Decision struct {
	Name    string
	Allowed bool
	Steps   []StepResult
	Roles   []RoleValue
}

// The answer always equals Resolve(s, o).Has(name), because both run the
// same decide.
func Explain(s Subject, o *Object, name string) Decision {
	d := &Decision{Name: name, Steps: make([]StepResult, len(Steps))}
	for i, step := range Steps {
		d.Steps[i] = StepResult{Step: step}
	}
	d.Allowed = prepare(s, o).decide(name, d)
	return *d
}

func (d *Decision) Step(step Step) StepResult { return d.Steps[step-1] }

func (d *Decision) ran(step Step) {
	if d != nil && d.Steps[step-1].Effect == EffectSkip {
		d.Steps[step-1].Effect = EffectNone
	}
}

func (d *Decision) add(step Step, c Cause) {
	if d == nil {
		return
	}
	at := &d.Steps[step-1]
	at.Causes = append(at.Causes, c)
	if c.Effect == EffectAllow || c.Effect == EffectDeny {
		at.Effect = c.Effect
	} else if at.Effect == EffectSkip {
		at.Effect = EffectNone
	}
}

// A refused name can be blocked at several steps. The first is reported as
// the cause and the rest come back too, so a fix can cover them all.
func (d Decision) Decisive() (StepResult, []StepResult) {
	if s := d.Step(StepAccount); s.Effect == EffectDeny {
		return s, nil
	}
	if s := d.Step(StepSuperuser); s.Effect == EffectAllow {
		return s, nil
	}
	roles := d.Step(StepRoles)
	if over := d.Step(StepOverrides); over.Effect == roles.Effect && (over.Effect == EffectAllow || over.Effect == EffectDeny) {
		roles = over
	}
	sequence := []StepResult{roles, d.Step(StepObject), d.Step(StepForum), d.Step(StepEmail), d.Step(StepSanctions)}
	last := -1
	for i, s := range sequence {
		if s.Effect == EffectAllow {
			last = i
		}
	}
	var blockers []StepResult
	for _, s := range sequence[last+1:] {
		if s.Effect == EffectDeny {
			blockers = append(blockers, s)
		}
	}
	switch {
	case d.Allowed:
		return sequence[last], nil
	case last < 0 && roles.Effect != EffectDeny:
		return roles, blockers
	case len(blockers) > 0:
		return blockers[0], blockers[1:]
	}
	return roles, nil
}

type preparedRole struct {
	id          int64
	base        map[string]Value
	override    map[string]Value
	hasOverride bool
}

func (r preparedRole) value(name string) Value {
	if v := r.override[name]; v != Unset {
		return v
	}
	return r.base[name]
}

type evaluator struct {
	s     Subject
	o     *Object
	roles []preparedRole
}

// A role reads only the first override carrying its own id, and within one
// role or one override a restriction outranks a permission.
func prepare(s Subject, o *Object) *evaluator {
	e := &evaluator{s: s, o: o, roles: make([]preparedRole, len(s.Roles))}
	for i, role := range s.Roles {
		p := preparedRole{id: role.ID, base: values(role.Permissions, role.Restrictions)}
		if o != nil {
			for _, override := range o.Overrides {
				if override.RoleID == role.ID {
					p.override = values(override.Permissions, override.Restrictions)
					p.hasOverride = true
					break
				}
			}
		}
		e.roles[i] = p
	}
	return e
}

func values(allow, deny []string) map[string]Value {
	out := make(map[string]Value, len(allow)+len(deny))
	for _, name := range allow {
		out[name] = Allow
	}
	for _, name := range deny {
		out[name] = Deny
	}
	return out
}

// A role that refuses a name refuses it for everyone holding that role, so a
// single refusal outweighs any number of grants.
func (e *evaluator) held(name string) bool {
	allowed := false
	for _, r := range e.roles {
		switch r.value(name) {
		case Deny:
			return false
		case Allow:
			allowed = true
		}
	}
	return allowed
}

func (e *evaluator) decide(name string, d *Decision) bool {
	s, o := e.s, e.o
	// An anonymous visitor is never inactive, so the flag does not apply.
	if !s.Active && !s.Anonymous {
		d.add(StepAccount, Cause{Reason: ReasonInactive, Effect: EffectDeny})
		return false
	}
	d.ran(StepAccount)
	if s.Active && s.Superuser {
		d.add(StepSuperuser, Cause{Reason: ReasonSuperuser, Effect: EffectAllow})
		return true
	}
	d.ran(StepSuperuser)

	v := e.held(name)
	if d != nil {
		e.traceRoles(name, v, d)
	}
	if o != nil {
		d.ran(StepObject)
		v = e.object(o, name, v, d)
	}
	d.ran(StepEmail)
	if s.Unverified && slices.Contains(unverified, name) {
		d.add(StepEmail, Cause{Reason: ReasonUnverified, Effect: EffectDeny})
		v = false
	}
	d.ran(StepSanctions)
	for _, kind := range s.Sanctions {
		if slices.Contains(sanctioned(kind), name) {
			d.add(StepSanctions, Cause{Reason: ReasonSanction, Effect: EffectDeny, Sanction: kind})
			v = false
		}
	}
	return v
}

func (e *evaluator) traceRoles(name string, held bool, d *Decision) {
	d.ran(StepRoles)
	baseAllowed, baseDenied, touched := false, false, false
	for _, r := range e.roles {
		rv := RoleValue{RoleID: r.id, Base: r.base[name], HasOverride: r.hasOverride, Override: r.override[name], Result: r.value(name)}
		d.Roles = append(d.Roles, rv)
		switch rv.Base {
		case Allow:
			baseAllowed = true
		case Deny:
			baseDenied = true
		}
		if rv.Override != Unset {
			touched = true
		}
	}

	want := Allow
	if !held {
		want = Deny
	}
	for _, rv := range d.Roles {
		if rv.Result == want {
			d.add(StepRoles, Cause{Reason: roleReason(want, rv.Override != Unset), Effect: effectOf(want), RoleID: rv.RoleID})
		}
	}
	if len(d.Step(StepRoles).Causes) == 0 {
		d.add(StepRoles, Cause{Reason: ReasonNoGrant, Effect: EffectNone})
	}

	if !slices.ContainsFunc(e.roles, func(r preparedRole) bool { return r.hasOverride }) {
		return
	}
	d.ran(StepOverrides)
	if !touched {
		return
	}
	effect := EffectNone
	if baseHeld := baseAllowed && !baseDenied; baseHeld != held {
		effect = effectOf(want)
	}
	for _, rv := range d.Roles {
		if rv.Override == Unset {
			continue
		}
		c := Cause{Reason: roleReason(rv.Override, true), Effect: EffectNone, RoleID: rv.RoleID}
		if rv.Override == want {
			c.Effect = effect
		}
		d.add(StepOverrides, c)
	}
}

func roleReason(v Value, override bool) Reason {
	switch {
	case v == Allow && override:
		return ReasonOverrideAllow
	case v == Allow:
		return ReasonRoleAllow
	case override:
		return ReasonOverrideDeny
	}
	return ReasonRoleDeny
}

func effectOf(v Value) Effect {
	if v == Allow {
		return EffectAllow
	}
	return EffectDeny
}

func (e *evaluator) object(o *Object, name string, v bool, d *Decision) bool {
	switch o.Kind {
	case KindForumSection:
		if name == ViewForumSections && o.HiddenForUsers && !e.held(ViewHiddenForumSections) {
			d.add(StepObject, Cause{Reason: ReasonHiddenSection, Effect: EffectDeny})
			return false
		}
		return v
	case KindForumThread:
		return e.thread(o, name, v, d)
	case KindForumPost:
		// The grant reads the site-wide answer rather than what this thread
		// has left, so a locked thread still lets its author keep the post.
		if name == EditForumPosts && o.Author && prepare(e.s, nil).decide(CreateForumPosts, nil) {
			d.add(StepObject, Cause{Reason: ReasonPostAuthor, Effect: EffectAllow})
			v = true
		}
		if o.Thread != nil {
			v = e.thread(o.Thread, name, v, d)
		}
		return v
	}
	if o.Locked {
		if !e.held(LockArticles) && slices.Contains(lockable, name) {
			d.add(StepObject, Cause{Reason: ReasonPageLocked, Effect: EffectDeny})
			return false
		}
		return v
	}
	if o.Author && name == ManageArticleAuthors {
		d.add(StepObject, Cause{Reason: ReasonPageAuthor, Effect: EffectAllow})
		return true
	}
	return v
}

func (e *evaluator) thread(t *Object, name string, v bool, d *Decision) bool {
	if t.Author && name == EditForumThreads {
		d.add(StepObject, Cause{Reason: ReasonThreadAuthor, Effect: EffectAllow})
		v = true
	}
	d.ran(StepForum)
	if !slices.Contains(silenced, name) {
		return v
	}
	if t.Locked && !e.held(LockForumThreads) {
		d.add(StepObject, Cause{Reason: ReasonThreadLocked, Effect: EffectDeny})
		v = false
	}
	if !e.s.Anonymous && !e.s.ForumActive {
		d.add(StepForum, Cause{Reason: ReasonForumBarred, Effect: EffectDeny})
		v = false
	}
	return v
}
