-- The addresses an account has been seen at, one row per pair rather than one
-- per action, so the table stays small enough to read whole.
CREATE TABLE pwikit_user_address (
    user_id    bigint      NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    address    inet        NOT NULL,
    first_seen timestamptz NOT NULL,
    last_seen  timestamptz NOT NULL,
    hits       integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, address)
);

-- Answering "who else came from here" is the whole point of the table.
CREATE INDEX pwikit_user_address_address_idx ON pwikit_user_address (address);

-- What staff did in the admin. Nothing else writes here, so it stays small and
-- is kept rather than pruned.
CREATE TABLE pwikit_admin_log (
    id         bigserial   PRIMARY KEY,
    user_id    bigint      REFERENCES web_user (id) ON DELETE SET NULL,
    stale_name text        NOT NULL DEFAULT '',
    action     text        NOT NULL,
    screen     text        NOT NULL,
    target     text        NOT NULL DEFAULT '',
    label      text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL
);

CREATE INDEX pwikit_admin_log_created_idx ON pwikit_admin_log (created_at DESC, id DESC);
