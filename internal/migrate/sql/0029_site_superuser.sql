-- compat: breaking
-- Accounts are shared by every site of an instance, so being a superuser of one
-- site cannot live on the account row the way the instance-wide flag does.
CREATE TABLE pwikit_site_superuser (
    site_id       bigint      NOT NULL REFERENCES web_site (id) ON DELETE CASCADE,
    user_id       bigint      NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    granted_by_id bigint      REFERENCES web_user (id) ON DELETE SET NULL,
    granted_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, user_id)
);

CREATE INDEX pwikit_site_superuser_user_idx ON pwikit_site_superuser (user_id);
