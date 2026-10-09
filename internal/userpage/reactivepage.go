package userpage

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/pageconfig"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	FavouritesPrefix = "/-/favourites"
	RatingsPrefix    = "/-/ratings"
	LikedPostsPrefix = "/-/liked-posts"
	MyTicketsPrefix  = "/-/my-tickets"

	NotificationsPrefix = "/-/notifications"
	MessagesPrefix      = "/-/messages"

	NotificationsSubPrefix = NotificationsPrefix + "/"
	MessagesSubPrefix      = MessagesPrefix + "/"
	MyTicketSubPrefix      = MyTicketsPrefix + "/"
)

var ticketKinds = []string{"ticket", "membershipapply", "report"}

var reactivePaths = []string{
	FavouritesPrefix,
	RatingsPrefix,
	LikedPostsPrefix,
	MyTicketsPrefix,
	NotificationsPrefix,
	NotificationsPrefix + "/all",
	NotificationsPrefix + "/unread",
	MessagesPrefix,
}

func answers(path string) bool {
	if slices.Contains(reactivePaths, path) {
		return true
	}
	if rest, ok := strings.CutPrefix(path, MyTicketSubPrefix); ok {
		kind, raw, ok := strings.Cut(rest, "/")
		if !ok || !slices.Contains(ticketKinds, kind) {
			return false
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		return err == nil && id > 0
	}
	rest, ok := strings.CutPrefix(path, MessagesSubPrefix)
	if !ok || rest == "" {
		return false
	}
	_, err := strconv.ParseInt(rest, 10, 64)
	return err == nil
}

type ReactiveHandler struct {
	deps Deps
}

var _ http.Handler = (*ReactiveHandler)(nil)

func NewReactive(d Deps) *ReactiveHandler { return &ReactiveHandler{deps: d} }

func (h *ReactiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if !answers(r.URL.Path) {
		loc := h.deps.Bundle.For(r.Context())
		shell.New(loc, h.deps.Assets).Missing(w, r, loc.T("system.page-not-found"))
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	viewer := auth.FromContext(ctx)
	if current == nil || viewer == nil {
		redirect(w, loginPath+"?to="+url.QueryEscape(r.URL.Path), http.StatusFound)
		return
	}

	body, err := h.page(r, current, viewer)
	if err != nil {
		h.deps.logger().Error("render reactive page", "path", r.URL.Path, "user", viewer.ID, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
}

func (h *ReactiveHandler) page(r *http.Request, current *db.Site, viewer *db.User) (string, error) {
	ctx := r.Context()
	loc := h.deps.Bundle.For(ctx)

	theme, err := site.ThemeURLByID(ctx, h.deps.DB, current.SystemThemeID)
	if err != nil {
		return "", err
	}
	config, err := h.config(r, viewer)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	err = shell.New(loc, h.deps.Assets).Reactive(&out, shell.Reactive{ThemeURL: theme, Config: config})
	return out.String(), err
}

func (h *ReactiveHandler) config(r *http.Request, viewer *db.User) (string, error) {
	ctx := r.Context()
	userRoles, err := h.deps.DB.RolesByUser(ctx, siteID(ctx), viewer.ID)
	if err != nil {
		return "", err
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(viewer, time.Now())
	if err != nil {
		return "", err
	}
	editor := perms.Resolve(subject, nil).Has(perms.EditArticles)
	user := pageconfig.SignedInUserJSON(viewer, userRoles, true, editor)
	reviews, err := repo.ReviewsTickets(ctx, h.deps.DB, viewer, time.Now())
	if err != nil {
		return "", err
	}
	if !reviews {
		submitted, err := h.deps.DB.OwnTicketCount(ctx, viewer.ID)
		if err != nil {
			return "", err
		}
		reviews = submitted > 0
	}
	out := wikijson.Object{{Key: "user", Value: user.Object()}, {Key: "reviewsTickets", Value: reviews}}
	if current := site.FromContext(ctx); current != nil {
		out = append(out, wikijson.Field{Key: "site", Value: wikijson.Object{{Key: "title", Value: current.Title}}})
	}
	return wikijson.Marshal(out)
}
