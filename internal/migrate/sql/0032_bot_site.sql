-- compat: breaking
-- Accounts are shared by every site, so the site a bot was made for cannot live
-- on the account row. Its key is shown only to that site's staff.
CREATE TABLE pwikit_bot_site (
    user_id bigint PRIMARY KEY REFERENCES web_user (id) ON DELETE CASCADE,
    site_id bigint NOT NULL REFERENCES web_site (id) ON DELETE CASCADE
);

CREATE INDEX pwikit_bot_site_site_idx ON pwikit_bot_site (site_id);

-- A bot whose creation was never logged stays without a site, and only an
-- instance superuser sees its key.
INSERT INTO pwikit_bot_site (user_id, site_id)
SELECT DISTINCT ON (u.id) u.id, l.site_id
FROM web_user u
JOIN pwikit_admin_log l ON l.screen = 'users' AND l.action = 'create'
    AND l.target = u.id::text AND l.site_id IS NOT NULL
WHERE u.type = 'bot'
ORDER BY u.id, l.created_at, l.id;
