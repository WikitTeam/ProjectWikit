-- compat: breaking
ALTER TABLE web_site ADD COLUMN language_follows_visitor boolean NOT NULL DEFAULT true;
