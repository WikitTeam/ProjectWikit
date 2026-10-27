-- compat: breaking
ALTER TABLE web_site ADD COLUMN signup_open boolean NOT NULL DEFAULT true;
