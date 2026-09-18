-- compat: breaking
-- A category can carry its own theme and navigation pages. An older build does
-- not read them and would dress the category in the site's instead.
ALTER TABLE web_category ADD COLUMN theme_id bigint REFERENCES web_theme (id);
ALTER TABLE web_category ADD COLUMN nav_top text NOT NULL DEFAULT '';
ALTER TABLE web_category ADD COLUMN nav_side text NOT NULL DEFAULT '';
