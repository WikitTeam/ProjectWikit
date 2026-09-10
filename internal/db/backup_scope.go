package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// how says what a table holds when one site is taken out on its own.
type how int

const (
	// owned rows belong to the site and come along, narrowed by where.
	owned how = iota
	// shared rows are the same on every instance, so the whole table comes.
	shared
	// people is the slice of accounts the exported rows point at.
	people
	// left rows belong to the operator or to users rather than to the site,
	// and stay behind.
	left
)

type rule struct {
	how   how
	where string
	// why is printed by the ratchet test when a table has no rule yet, and
	// read by anyone auditing what a site export hands over.
	why string
}

// The names bound before each query. Written once here so a rule reads as one
// line rather than as a paragraph of subqueries.
const scopeCTE = `
WITH s AS (SELECT id FROM web_site WHERE slug = $1),
     art AS (SELECT id FROM web_article WHERE site_id = (SELECT id FROM s)),
     cat AS (SELECT id FROM web_category WHERE site_id = (SELECT id FROM s)),
     rol AS (SELECT id FROM web_role WHERE site_id = (SELECT id FROM s)),
     sec AS (SELECT id FROM web_forumsection WHERE site_id = (SELECT id FROM s)),
     fcat AS (SELECT id FROM web_forumcategory WHERE section_id IN (SELECT id FROM sec)),
     thr AS (SELECT id FROM web_forumthread WHERE site_id = (SELECT id FROM s)),
     pst AS (SELECT id FROM web_forumpost WHERE thread_id IN (SELECT id FROM thr)),
     ovr AS (SELECT rolepermissionsoverride_id AS id FROM web_category_permissions_override
             WHERE category_id IN (SELECT id FROM cat))
`

// peopleQuery gathers every account the exported rows point at. Nothing decides
// up front who counts; the references decide, which is why no key can dangle.
const peopleQuery = scopeCTE + `,
     who AS (
       SELECT user_id AS id FROM web_article_authors WHERE article_id IN (SELECT id FROM art)
       UNION SELECT user_id FROM web_articlefavourite WHERE article_id IN (SELECT id FROM art)
       UNION SELECT user_id FROM web_articlelogentry WHERE article_id IN (SELECT id FROM art)
       UNION SELECT user_id FROM web_vote WHERE article_id IN (SELECT id FROM art)
       UNION SELECT author_id FROM web_file WHERE article_id IN (SELECT id FROM art)
       UNION SELECT deleted_by_id FROM web_file WHERE article_id IN (SELECT id FROM art)
       UNION SELECT author_id FROM web_forumthread WHERE site_id = (SELECT id FROM s)
       UNION SELECT author_id FROM web_forumpost WHERE thread_id IN (SELECT id FROM thr)
       UNION SELECT author_id FROM web_forumpostversion WHERE post_id IN (SELECT id FROM pst)
       UNION SELECT user_id FROM web_forumpostlike WHERE post_id IN (SELECT id FROM pst)
       UNION SELECT subscriber_id FROM web_usernotificationsubscription
             WHERE article_id IN (SELECT id FROM art) OR forum_thread_id IN (SELECT id FROM thr)
       UNION SELECT user_id FROM web_user_roles WHERE role_id IN (SELECT id FROM rol)
       UNION SELECT created_by_id FROM web_invitelink WHERE site_id = (SELECT id FROM s)
       UNION SELECT target_id FROM web_invitelink WHERE site_id = (SELECT id FROM s)
       UNION SELECT reporter_id FROM web_userreport WHERE site_id = (SELECT id FROM s)
       UNION SELECT reported_id FROM web_userreport WHERE site_id = (SELECT id FROM s)
       UNION SELECT reviewed_by_id FROM web_userreport WHERE site_id = (SELECT id FROM s)
       UNION SELECT author_id FROM web_userticket WHERE site_id = (SELECT id FROM s)
       UNION SELECT reviewed_by_id FROM web_userticket WHERE site_id = (SELECT id FROM s)
       UNION SELECT user_id FROM pwikit_admin_log WHERE site_id = (SELECT id FROM s)
     )
SELECT id FROM who WHERE id IS NOT NULL`

// A column that names something staying behind cannot travel as it is.
var swaps = map[string]map[string]string{
	"web_externallink": {
		"to_site_id": `CASE WHEN t.to_site_id = (SELECT id FROM s) THEN t.to_site_id END`,
	},
}

var rules = map[string]rule{
	"web_site":     {owned, `id = (SELECT id FROM s)`, ""},
	"web_settings": {owned, `site_id = (SELECT id FROM s)`, ""},

	"web_article":            {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_article_authors":    {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_article_tags":       {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_articlefavourite":   {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_articlelogentry":    {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_articlesearchindex": {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_articleversion":     {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_file":               {owned, `article_id IN (SELECT id FROM art)`, ""},
	"web_vote":               {owned, `article_id IN (SELECT id FROM art)`, ""},

	"web_category":                      {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_category_permissions_override": {owned, `category_id IN (SELECT id FROM cat)`, ""},
	"web_tag":                           {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_tagscategory":                  {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_theme":                         {owned, `site_id = (SELECT id FROM s)`, ""},

	"web_role":                                {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_rolecategory":                        {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_role_permissions":                    {owned, `role_id IN (SELECT id FROM rol)`, ""},
	"web_role_restrictions":                   {owned, `role_id IN (SELECT id FROM rol)`, ""},
	"web_user_roles":                          {owned, `role_id IN (SELECT id FROM rol)`, ""},
	"web_rolepermissionsoverride":             {owned, `id IN (SELECT id FROM ovr)`, ""},
	"web_rolepermissionsoverride_permissions": {owned, `rolepermissionsoverride_id IN (SELECT id FROM ovr)`, ""},
	"web_rolepermissionsoverride_restrictions": {owned,
		`rolepermissionsoverride_id IN (SELECT id FROM ovr)`, ""},

	"web_forumsection":     {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_forumcategory":    {owned, `section_id IN (SELECT id FROM sec)`, ""},
	"web_forumthread":      {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_forumpost":        {owned, `thread_id IN (SELECT id FROM thr)`, ""},
	"web_forumpostversion": {owned, `post_id IN (SELECT id FROM pst)`, ""},
	"web_forumpostlike":    {owned, `post_id IN (SELECT id FROM pst)`, ""},

	"web_invitelink":   {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_userreport":   {owned, `site_id = (SELECT id FROM s)`, ""},
	"web_userticket":   {owned, `site_id = (SELECT id FROM s)`, ""},
	"pwikit_admin_log": {owned, `site_id = (SELECT id FROM s)`, ""},

	// A subscription can hang off an article or off a forum thread, and the
	// thread need not belong to an article.
	"web_usernotificationsubscription": {owned,
		`article_id IN (SELECT id FROM art) OR forum_thread_id IN (SELECT id FROM thr)`, ""},

	// A link whose other end stays behind is still worth keeping, since
	// forgetting which site that was is the same thing as the link being red.
	"web_externallink": {owned, `from_site_id = (SELECT id FROM s)`, ""},

	"web_user": {people, "", ""},
	"dynamic_preferences_users_userpreferencemodel": {people, "", ""},

	// The baseline seeds these and every instance holds the same rows, but the
	// roles point at them, so they have to travel.
	"auth_permission":     {shared, "", ""},
	"django_content_type": {shared, "", ""},
	"django_migrations":   {shared, "", ""},

	"web_directmessage":                         {left, "", "private mail belongs to the two people, not to a site"},
	"web_directmessageblock":                    {left, "", "a block follows the person across every site"},
	"web_usernotification":                      {left, "", "a notice can point at a page on a site that is staying"},
	"web_usernotificationmapping":               {left, "", "it hangs off a notice that is staying"},
	"pwikit_user_address":                       {left, "", "sign-in addresses are what the operator watches, not site content"},
	"web_usedtoken":                             {left, "", "a spent token is worth nothing anywhere"},
	"web_actionlogentry":                        {left, "", "nothing writes it any more"},
	"django_session":                            {left, "", "a session belongs to the server it was opened on"},
	"django_admin_log":                          {left, "", "nothing writes it any more"},
	"auth_group":                                {left, "", "roles took over from groups"},
	"auth_group_permissions":                    {left, "", "roles took over from groups"},
	"web_user_groups":                           {left, "", "roles took over from groups"},
	"web_user_user_permissions":                 {left, "", "roles took over from per-account rights"},
	"dynamic_preferences_globalpreferencemodel": {left, "", "it belongs to the instance"},
}

// An empty return means the table does not travel.
func SiteExportQuery(ctx context.Context, conn *pgx.Conn, table string, keepPasswords bool) (string, error) {
	r, ok := rules[table]
	if !ok {
		return "", fmt.Errorf("no rule says whether %s belongs to a site; add one to internal/db/backup_scope.go", table)
	}
	switch r.how {
	case left:
		return "", nil
	case shared:
		columns, err := columnList(ctx, conn, table, nil)
		if err != nil {
			return "", err
		}
		return `SELECT ` + columns + ` FROM ` + QuoteName(table) + ` t`, nil
	case people:
		return peopleFor(ctx, conn, table, keepPasswords)
	default:
		columns, err := columnList(ctx, conn, table, swaps[table])
		if err != nil {
			return "", err
		}
		return scopeCTE + `SELECT ` + columns + ` FROM ` + QuoteName(table) + ` t WHERE ` + r.where, nil
	}
}

func peopleFor(ctx context.Context, conn *pgx.Conn, table string, keepPasswords bool) (string, error) {
	if table != "web_user" {
		columns, err := columnList(ctx, conn, table, nil)
		if err != nil {
			return "", err
		}
		return `WITH who AS (` + peopleQuery + `) SELECT ` + columns + ` FROM ` + QuoteName(table) +
			` t WHERE t.instance_id IN (SELECT id FROM who)`, nil
	}
	// Whoever runs the site now decides who administers it, and a key issued on
	// the old instance should not open the new one.
	swap := map[string]string{"is_superuser": `false`, "api_key": `NULL`}
	if !keepPasswords {
		// The marker every comparison fails against. The account arrives, but
		// nobody signs in as it until they reset.
		swap["password"] = `'!'`
	}
	columns, err := columnList(ctx, conn, table, swap)
	if err != nil {
		return "", err
	}
	return `WITH who AS (` + peopleQuery + `) SELECT ` + columns +
		` FROM web_user t WHERE t.id IN (SELECT id FROM who)`, nil
}

// A generated column is left out, because a plain COPY of the table leaves it
// out too and a restore has to see the same shape either way.
func columnList(ctx context.Context, conn *pgx.Conn, table string, swap map[string]string) (string, error) {
	rows, err := conn.Query(ctx, `
SELECT a.attname
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname = $1 AND a.attnum > 0 AND NOT a.attisdropped
  AND a.attgenerated = ''
ORDER BY a.attnum`, table)
	if err != nil {
		return "", fmt.Errorf("read the columns of %s: %w", table, err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if with, ok := swap[name]; ok {
			parts = append(parts, with+" AS "+QuoteName(name))
			continue
		}
		parts = append(parts, "t."+QuoteName(name))
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(parts, ", "), nil
}

// SiteScopeRuled lets a test hold the schema and this file to each other.
func SiteScopeRuled() []string {
	out := make([]string, 0, len(rules))
	for name := range rules {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
