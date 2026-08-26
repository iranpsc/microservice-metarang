-- Persist whether the session that issued a personal access token used crypto-wallet login.
-- Apply to existing databases that were created from an older schema.sql.

-- migrate:up
ALTER TABLE `personal_access_tokens`
  ADD COLUMN `wallet_login` tinyint(1) NOT NULL DEFAULT 0 AFTER `expires_at`;

-- migrate:down
ALTER TABLE `personal_access_tokens`
  DROP COLUMN `wallet_login`;
