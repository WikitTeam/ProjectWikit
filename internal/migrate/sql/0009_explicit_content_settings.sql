-- A site has nothing above it to follow, so its own rows name a value. The
-- values written here are the ones the code fell back to. Categories keep
-- following the site, which is still an answer they can give.
UPDATE web_settings SET rating_mode = 'updown'
WHERE site_id IS NOT NULL AND rating_mode = 'default';

UPDATE web_settings SET can_user_create_tags = 'disabled'
WHERE site_id IS NOT NULL AND can_user_create_tags = 'default';
