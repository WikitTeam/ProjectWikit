-- compat: breaking
CREATE TABLE pwikit_ip_ban (
    id        bigserial   PRIMARY KEY,
    site_id   bigint      NOT NULL REFERENCES web_site (id) ON DELETE CASCADE,
    network   cidr        NOT NULL,
    until     timestamptz,
    reason    text        NOT NULL DEFAULT '',
    set_by_id bigint      REFERENCES web_user (id) ON DELETE SET NULL,
    set_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (site_id, network)
);
