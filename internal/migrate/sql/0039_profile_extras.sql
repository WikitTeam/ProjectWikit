-- compat: compatible
CREATE TABLE pwikit_profile_field (
    user_id  bigint  NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    position integer NOT NULL,
    label    text    NOT NULL,
    value    text    NOT NULL,
    PRIMARY KEY (user_id, position)
);

CREATE TABLE pwikit_profile_link (
    user_id  bigint  NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    position integer NOT NULL,
    platform text    NOT NULL,
    value    text    NOT NULL,
    PRIMARY KEY (user_id, position)
);

CREATE TABLE pwikit_profile_pin (
    user_id    bigint      NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    article_id bigint      NOT NULL REFERENCES web_article (id) ON DELETE CASCADE,
    pinned_at  timestamptz NOT NULL,
    PRIMARY KEY (user_id, article_id)
);
