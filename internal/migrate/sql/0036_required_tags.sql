-- compat: breaking
CREATE TABLE pwikit_category_tag_rule (
    category_id bigint PRIMARY KEY REFERENCES web_category (id) ON DELETE CASCADE,
    groups      text   NOT NULL DEFAULT ''
);

CREATE TABLE pwikit_category_tag_exempt (
    category_id bigint NOT NULL REFERENCES web_category (id) ON DELETE CASCADE,
    role_id     bigint NOT NULL REFERENCES web_role (id) ON DELETE CASCADE,
    PRIMARY KEY (category_id, role_id)
);
