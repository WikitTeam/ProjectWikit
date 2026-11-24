-- compat: breaking
-- A release without this table would let a site past the limits set for it.
CREATE TABLE pwikit_site_upload_limit (
    site_id    bigint PRIMARY KEY REFERENCES web_site (id) ON DELETE CASCADE,
    max_file   bigint,
    site_quota bigint
);
