-- What a site has done to one of its members. Accounts are shared by every
-- site of an instance, so a sanction that belongs to one site cannot live on
-- the account row.
CREATE TABLE pwikit_member_sanction (
    id        bigserial   PRIMARY KEY,
    site_id   bigint      NOT NULL REFERENCES web_site (id) ON DELETE CASCADE,
    user_id   bigint      NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    kind      text        NOT NULL,
    -- Null lasts until someone lifts it.
    until     timestamptz,
    reason    text        NOT NULL DEFAULT '',
    set_by_id bigint      REFERENCES web_user (id) ON DELETE SET NULL,
    set_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (site_id, user_id, kind)
);

CREATE INDEX pwikit_member_sanction_site_idx ON pwikit_member_sanction (site_id, until);

INSERT INTO auth_permission (id, name, content_type_id, codename)
SELECT (SELECT max(id) FROM auth_permission) + row_number() OVER (ORDER BY wanted.codename),
       '', content.id, wanted.codename
FROM (VALUES
    ('ban_members'),
    ('mute_members'),
    ('restrict_member_editing'),
    ('restrict_member_rating'),
    ('reset_member_votes'),
    ('invite_members'),
    ('manage_bots')
) AS wanted (codename)
CROSS JOIN (
    SELECT id FROM django_content_type WHERE app_label = 'web' AND model = 'roles'
) AS content
WHERE NOT EXISTS (SELECT 1 FROM auth_permission p WHERE p.codename = wanted.codename);

SELECT setval(pg_get_serial_sequence('auth_permission', 'id'), (SELECT max(id) FROM auth_permission));

-- Whoever managed members before this split keeps everything the split created,
-- so an upgrade takes nothing away.
INSERT INTO web_role_permissions (role_id, permission_id)
SELECT held.role_id, added.id
FROM web_role_permissions held
JOIN auth_permission manage ON manage.id = held.permission_id AND manage.codename = 'manage_users'
CROSS JOIN auth_permission added
WHERE added.codename IN ('ban_members', 'mute_members', 'restrict_member_editing',
                         'restrict_member_rating', 'reset_member_votes', 'invite_members', 'manage_bots')
  AND NOT EXISTS (
      SELECT 1 FROM web_role_permissions have
      WHERE have.role_id = held.role_id AND have.permission_id = added.id);
