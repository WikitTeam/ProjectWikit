package pageconfig

import (
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/pyjson"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

const (
	typeAnonymous = "anonymous"
	typeSystem    = "system"
)

type UserJSON struct {
	Type       string
	ID         any
	Name       any
	Username   any
	URLName    any
	IsActive   bool
	Avatar     any
	ShowAvatar bool
	Admin      bool
	Staff      bool
	Editor     any
	Roles      any
}

func (u UserJSON) object() pyjson.Object {
	return pyjson.Object{
		{Key: "type", Value: u.Type},
		{Key: "id", Value: u.ID},
		{Key: "name", Value: u.Name},
		{Key: "username", Value: u.Username},
		{Key: "urlName", Value: u.URLName},
		{Key: "isActive", Value: u.IsActive},
		{Key: "avatar", Value: u.Avatar},
		{Key: "showAvatar", Value: u.ShowAvatar},
		{Key: "admin", Value: u.Admin},
		{Key: "staff", Value: u.Staff},
		{Key: "editor", Value: u.Editor},
		{Key: "roles", Value: u.Roles},
	}
}

func (u UserJSON) JSON() (string, error) { return pyjson.Marshal(u.object()) }

func SystemUserJSON() UserJSON {
	return UserJSON{Type: typeSystem, IsActive: true, Editor: false}
}

func AnonymousUserJSON(loc *i18n.Localizer, showAvatar bool) UserJSON {
	return UserJSON{
		Type:       typeAnonymous,
		Name:       loc.T("user-anonymous"),
		IsActive:   true,
		ShowAvatar: showAvatar,
		Editor:     false,
	}
}

func SignedInUserJSON(u *db.User, userRoles []roles.Role, showAvatar, editor bool) UserJSON {
	out := UserJSON{
		Type:       u.Type,
		ID:         u.ID,
		Name:       u.DisplayLabel(),
		Username:   u.Username,
		URLName:    u.URLName(),
		IsActive:   u.IsActive,
		ShowAvatar: showAvatar,
		Admin:      u.IsSuperuser,
		Staff:      IsStaff(u, userRoles),
		Editor:     editor,
		Roles:      visualSlugs(userRoles),
	}
	if u.Avatar != "" {
		out.Avatar = "/local--files/" + u.Avatar
	}
	return out
}

func IsStaff(u *db.User, userRoles []roles.Role) bool {
	if u.IsSuperuser {
		return true
	}
	for _, role := range userRoles {
		if role.IsStaff {
			return true
		}
	}
	return false
}

func visualSlugs(userRoles []roles.Role) []string {
	out := []string{}
	for _, role := range userRoles {
		if role.IsVisual() {
			out = append(out, role.Slug)
		}
	}
	return out
}

type LoginStatus struct {
	User              *db.User
	Roles             []roles.Role
	CanEditArticles   bool
	NotificationCount int
}

func (l LoginStatus) JSON(loc *i18n.Localizer) (string, error) {
	user := AnonymousUserJSON(loc, true)
	if l.User != nil {
		user = SignedInUserJSON(l.User, l.Roles, true, l.CanEditArticles)
	}
	return pyjson.Marshal(pyjson.Object{
		{Key: "user", Value: user.object()},
		{Key: "notificationCount", Value: l.NotificationCount},
	})
}
