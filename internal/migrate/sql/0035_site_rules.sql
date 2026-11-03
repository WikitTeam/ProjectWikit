-- compat: breaking
ALTER TABLE web_site ADD COLUMN self_votes_forbidden boolean NOT NULL DEFAULT false;
ALTER TABLE web_site ADD COLUMN edit_forbidden_hint text NOT NULL DEFAULT '';
ALTER TABLE web_site ADD COLUMN deletion_notice_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE web_site ADD COLUMN deletion_notice text NOT NULL DEFAULT '';
