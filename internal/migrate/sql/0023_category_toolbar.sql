-- compat: breaking
CREATE TABLE pwikit_category_toolbar_hidden (
    category_id bigint NOT NULL REFERENCES web_category (id) ON DELETE CASCADE,
    role_id     bigint NOT NULL REFERENCES web_role (id) ON DELETE CASCADE,
    PRIMARY KEY (category_id, role_id)
);
