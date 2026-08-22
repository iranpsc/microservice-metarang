-- Unique constraint for one active profile limitation per limiter/limited pair.
-- Apply to existing databases that were created from an older schema.sql.

-- migrate:up
ALTER TABLE `profile_limitations`
  ADD UNIQUE KEY `profile_limitations_limiter_limited_unique` (`limiter_user_id`, `limited_user_id`);

-- migrate:down
ALTER TABLE `profile_limitations`
  DROP INDEX `profile_limitations_limiter_limited_unique`;
