-- compat: breaking
CREATE TABLE pwikit_message_clear (
    user_id    bigint NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    partner_id bigint NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    through_id bigint NOT NULL,
    PRIMARY KEY (user_id, partner_id)
);
