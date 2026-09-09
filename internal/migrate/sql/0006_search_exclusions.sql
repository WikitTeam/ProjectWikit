-- web_category already carried this flag and the admin already offered it, but
-- nothing read it. A tag and a page get the same switch.
ALTER TABLE web_tag ADD COLUMN is_indexed boolean NOT NULL DEFAULT true;
ALTER TABLE web_article ADD COLUMN is_indexed boolean NOT NULL DEFAULT true;
