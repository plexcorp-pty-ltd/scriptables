-- Scriptables now runs natively and authenticates with the SSH keys already
-- present on the machine (~/.ssh plus any running ssh-agent), so keys are no
-- longer stored in the database.
--
-- Safe to run against a fresh database built from 001_setup_db.sql (where these
-- objects never existed) as well as an install predating this change.
ALTER TABLE `servers` DROP COLUMN IF EXISTS `ssh_key_id`;
ALTER TABLE `sites` DROP COLUMN IF EXISTS `ssh_key_id`;
DROP TABLE IF EXISTS `ssh_keys`;
