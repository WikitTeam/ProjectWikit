-- compat: compatible
CREATE TABLE pwikit_update (
    id                  smallint    PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    checked_at          timestamptz,
    check_error         text        NOT NULL DEFAULT '',
    next_check_at       timestamptz,
    latest_version      text        NOT NULL DEFAULT '',
    latest_published_at timestamptz,
    latest_postgres     text        NOT NULL DEFAULT '',
    latest_notes        text        NOT NULL DEFAULT '',
    scheduled_version   text        NOT NULL DEFAULT '',
    scheduled_at        timestamptz,
    postponed_until     timestamptz,
    skipped_version     text        NOT NULL DEFAULT '',
    failed_versions     text[]      NOT NULL DEFAULT '{}',
    pinned_version      text        NOT NULL DEFAULT '',
    last_from           text        NOT NULL DEFAULT '',
    last_to             text        NOT NULL DEFAULT '',
    last_outcome        text        NOT NULL DEFAULT '',
    last_error          text        NOT NULL DEFAULT '',
    last_at             timestamptz,
    rollback_version    text        NOT NULL DEFAULT '',
    rollback_kind       text        NOT NULL DEFAULT '',
    rollback_expires_at timestamptz
);

INSERT INTO pwikit_update (id) VALUES (1);
