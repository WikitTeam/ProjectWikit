-- compat: compatible
ALTER TABLE web_user ADD COLUMN held_email text NOT NULL DEFAULT '';
ALTER TABLE web_user ADD COLUMN held_until timestamptz;
ALTER TABLE web_user ADD COLUMN held_site_id bigint REFERENCES web_site (id) ON DELETE SET NULL;

CREATE INDEX pwikit_user_held_idx ON web_user (held_until) WHERE held_until IS NOT NULL;
