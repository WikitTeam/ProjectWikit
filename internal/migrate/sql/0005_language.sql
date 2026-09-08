ALTER TABLE web_site ADD COLUMN language text NOT NULL DEFAULT 'zh-hans';

-- Empty says the member never chose, which is what leaves the browser a say.
ALTER TABLE web_user ADD COLUMN language text NOT NULL DEFAULT '';
