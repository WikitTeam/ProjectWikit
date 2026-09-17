-- compat: compatible
-- Two sites can hold a page of the same name linking to the same target, and
-- the older key let the second site's link be dropped without a word.
ALTER TABLE web_externallink DROP CONSTRAINT web_externallink_unique;
ALTER TABLE web_externallink ADD CONSTRAINT web_externallink_unique
    UNIQUE (from_site_id, link_from, link_type, to_site_id, link_to);
