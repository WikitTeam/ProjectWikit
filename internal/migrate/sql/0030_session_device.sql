-- compat: compatible
CREATE TABLE pwikit_session_device (
    id bigserial PRIMARY KEY,
    session_key varchar(40) NOT NULL UNIQUE REFERENCES django_session (session_key) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    site_id bigint REFERENCES web_site (id) ON DELETE SET NULL,
    signed_in_at timestamptz,
    seen_at timestamptz NOT NULL,
    address inet,
    agent text NOT NULL DEFAULT ''
);
CREATE INDEX pwikit_session_device_user_idx ON pwikit_session_device (user_id);
