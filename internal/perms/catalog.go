package perms

type Group struct {
	Key   string
	Names []string
}

var Catalog = []Group{
	{Key: "articles", Names: []string{
		"view_articles",
		"rate_articles",
		"create_articles",
		"edit_articles",
		"tag_articles",
		"move_articles",
		"lock_articles",
		"manage_article_files",
		"delete_articles",
		"reset_article_votes",
		"comment_articles",
		"view_article_comments",
		"manage_article_authors",
	}},
	{Key: "forum", Names: []string{
		"view_forum_posts",
		"create_forum_posts",
		"edit_forum_posts",
		"delete_forum_posts",
		"view_forum_threads",
		"create_forum_threads",
		"edit_forum_threads",
		"pin_forum_threads",
		"lock_forum_threads",
		"move_forum_threads",
		"view_forum_sections",
		"view_hidden_forum_sections",
		"view_forum_categories",
	}},
	{Key: "social", Names: []string{
		"send_direct_message",
	}},
	{Key: "tickets", Names: []string{
		"view_user_reports",
		"view_reported_full_conversation",
		"view_user_tickets",
		"review_membership_applications",
	}},
	{Key: "admin", Names: []string{
		"manage_users",
		"manage_roles",
		"manage_site",
		"view_actions_log",
		"manage_categories",
		"manage_tags",
		"manage_forum",
		"view_sensitive_info",
		"view_votes_timestamp",
		"manage_updates",
		"manage_permissions",
	}},
}

func GroupOf(name string) string {
	for _, g := range Catalog {
		for _, n := range g.Names {
			if n == name {
				return g.Key
			}
		}
	}
	return ""
}
