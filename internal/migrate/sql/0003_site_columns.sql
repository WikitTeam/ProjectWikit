ALTER TABLE web_user ADD COLUMN wikidot_user_id bigint;

-- The archive keys every author and voter by this number while usernames drift,
-- so it is the only stable way to recognise someone across two backups.
CREATE UNIQUE INDEX web_user_wikidot_user_id_key ON web_user (wikidot_user_id);

ALTER TABLE web_article ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_category ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_tag ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_tagscategory ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_role ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_rolecategory ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_forumsection ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_theme ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_invitelink ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_userreport ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE web_userticket ADD COLUMN site_id bigint REFERENCES web_site (id);
ALTER TABLE pwikit_admin_log ADD COLUMN site_id bigint REFERENCES web_site (id);

-- A comment thread hangs off an article and a forum thread off a category, so
-- without a column of its own the site takes a two-way branch to answer.
ALTER TABLE web_forumthread ADD COLUMN site_id bigint REFERENCES web_site (id);

DO $$
DECLARE
    only_site bigint;
    one text;
BEGIN
    SELECT id INTO only_site FROM web_site ORDER BY id LIMIT 1;
    IF only_site IS NULL THEN
        RETURN;
    END IF;
    FOREACH one IN ARRAY ARRAY[
        'web_article', 'web_category', 'web_tag', 'web_tagscategory', 'web_role',
        'web_rolecategory', 'web_forumsection', 'web_forumthread', 'web_theme',
        'web_invitelink', 'web_userreport', 'web_userticket', 'pwikit_admin_log'
    ] LOOP
        EXECUTE format('UPDATE %I SET site_id = %s', one, only_site);
    END LOOP;
END $$;

CREATE INDEX web_article_site_idx ON web_article (site_id);
CREATE INDEX web_forumthread_site_idx ON web_forumthread (site_id);

ALTER TABLE web_article DROP CONSTRAINT web_article_unique;
ALTER TABLE web_article ADD CONSTRAINT web_article_unique UNIQUE (site_id, category, name);

ALTER TABLE web_category DROP CONSTRAINT web_category_unique;
ALTER TABLE web_category ADD CONSTRAINT web_category_unique UNIQUE (site_id, name);

ALTER TABLE web_role DROP CONSTRAINT web_role_slug_key;
ALTER TABLE web_role ADD CONSTRAINT web_role_slug_key UNIQUE (site_id, slug);

ALTER TABLE web_tag DROP CONSTRAINT web_tag_unique;
ALTER TABLE web_tag ADD CONSTRAINT web_tag_unique UNIQUE (site_id, category_id, name);

-- Two constraints said the same thing about the slug, and only one of them
-- comes back.
ALTER TABLE web_tagscategory DROP CONSTRAINT web_tagscategory_unique;
ALTER TABLE web_tagscategory DROP CONSTRAINT web_tagscategory_slug_key;
ALTER TABLE web_tagscategory ADD CONSTRAINT web_tagscategory_unique UNIQUE (site_id, slug);

ALTER TABLE web_tagscategory DROP CONSTRAINT web_tagscategory_priority_fd2df012_uniq;
ALTER TABLE web_tagscategory ADD CONSTRAINT web_tagscategory_priority_fd2df012_uniq UNIQUE (site_id, priority);

ALTER TABLE web_theme DROP CONSTRAINT web_theme_slug_7893de0e_uniq;
ALTER TABLE web_theme ADD CONSTRAINT web_theme_slug_7893de0e_uniq UNIQUE (site_id, slug);

-- Host routing reads both columns as one namespace. Without this a second site
-- can claim the first site's media domain, which puts uploaded HTML on an
-- origin that is not its own.
ALTER TABLE web_site ADD CONSTRAINT web_site_media_domain_unique UNIQUE (media_domain);
