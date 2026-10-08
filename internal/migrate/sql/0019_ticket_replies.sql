-- compat: compatible
ALTER TABLE web_userticket ADD COLUMN reply text NOT NULL DEFAULT '';
ALTER TABLE web_userreport ADD COLUMN reply text NOT NULL DEFAULT '';
