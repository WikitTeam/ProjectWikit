-- compat: compatible
ALTER TABLE pwikit_update ADD COLUMN scheduled_by_hand boolean NOT NULL DEFAULT false;
