-- compat: compatible
CREATE TABLE pwikit_vote_event (
    id         bigserial        PRIMARY KEY,
    article_id bigint           NOT NULL REFERENCES web_article (id) ON DELETE CASCADE,
    user_id    bigint           REFERENCES web_user (id) ON DELETE SET NULL,
    old_rate   double precision,
    new_rate   double precision,
    at         timestamptz      NOT NULL
);

CREATE INDEX pwikit_vote_event_article_idx ON pwikit_vote_event (article_id, at, id);
