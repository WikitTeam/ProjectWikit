-- compat: compatible
CREATE TABLE pwikit_membership_quiz (
    site_id        bigint  PRIMARY KEY REFERENCES web_site (id) ON DELETE CASCADE,
    enabled        boolean NOT NULL DEFAULT false,
    role_id        bigint  REFERENCES web_role (id) ON DELETE SET NULL,
    tries          integer NOT NULL DEFAULT 3,
    cooldown_hours integer NOT NULL DEFAULT 12,
    questions      text    NOT NULL DEFAULT ''
);

CREATE TABLE pwikit_membership_quiz_attempt (
    site_id bigint      NOT NULL REFERENCES web_site (id) ON DELETE CASCADE,
    user_id bigint      NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    at      timestamptz NOT NULL,
    passed  boolean     NOT NULL
);

CREATE INDEX pwikit_membership_quiz_attempt_user_idx ON pwikit_membership_quiz_attempt (site_id, user_id, at);
