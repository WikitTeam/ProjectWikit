-- compat: breaking
CREATE TABLE pwikit_notify_mute (
    id         bigserial   PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    article_id bigint      REFERENCES web_article (id) ON DELETE CASCADE,
    thread_id  bigint      REFERENCES web_forumthread (id) ON DELETE CASCADE,
    muted_user bigint      REFERENCES web_user (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    CHECK (num_nonnulls(article_id, thread_id, muted_user) = 1)
);

CREATE UNIQUE INDEX pwikit_notify_mute_article_idx ON pwikit_notify_mute (user_id, article_id) WHERE article_id IS NOT NULL;
CREATE UNIQUE INDEX pwikit_notify_mute_thread_idx ON pwikit_notify_mute (user_id, thread_id) WHERE thread_id IS NOT NULL;
CREATE UNIQUE INDEX pwikit_notify_mute_user_idx ON pwikit_notify_mute (user_id, muted_user) WHERE muted_user IS NOT NULL;
