-- compat: compatible
CREATE TABLE pwikit_plugin_site (
    site_id bigint NOT NULL REFERENCES web_site(id) ON DELETE CASCADE,
    plugin text NOT NULL,
    enabled_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, plugin)
);

CREATE TABLE pwikit_plugin_kv (
    site_id bigint NOT NULL REFERENCES web_site(id) ON DELETE CASCADE,
    plugin text NOT NULL,
    key text NOT NULL,
    value bytea NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, plugin, key)
);
