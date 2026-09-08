ALTER TABLE web_externallink ADD COLUMN from_site_id bigint REFERENCES web_site (id);

-- An include may name a page on another site, so the two ends of a reference do
-- not have to agree on the site.
ALTER TABLE web_externallink ADD COLUMN to_site_id bigint REFERENCES web_site (id);

DO $$
DECLARE
    only_site bigint;
BEGIN
    SELECT id INTO only_site FROM web_site ORDER BY id LIMIT 1;
    IF only_site IS NULL THEN
        RETURN;
    END IF;
    UPDATE web_externallink SET from_site_id = only_site, to_site_id = only_site;
END $$;

CREATE INDEX web_externallink_to_idx ON web_externallink (to_site_id, link_to);
CREATE INDEX web_externallink_from_idx ON web_externallink (from_site_id, link_from);
