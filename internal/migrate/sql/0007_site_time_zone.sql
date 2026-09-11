-- UTC until somebody picks one, since the zone of the machine says nothing about
-- where the readers of a site are.
ALTER TABLE web_site ADD COLUMN time_zone text NOT NULL DEFAULT 'UTC';
