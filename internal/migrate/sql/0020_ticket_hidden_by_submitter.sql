-- compat: compatible
ALTER TABLE web_userticket ADD COLUMN hidden_by_author_at timestamptz;
ALTER TABLE web_userreport ADD COLUMN hidden_by_reporter_at timestamptz;
