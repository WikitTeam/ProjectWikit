-- compat: breaking
ALTER TABLE web_site ADD COLUMN application_role_id bigint REFERENCES web_role (id) ON DELETE SET NULL;

UPDATE web_site s SET application_role_id = r.id
FROM web_role r
WHERE r.site_id = s.id AND r.slug = 'member';
