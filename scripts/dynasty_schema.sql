-- Dynasty Service Database Schema
-- This script creates all tables required for the dynasty-service

-- Create dynasties table
CREATE TABLE IF NOT EXISTS `dynasties` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `feature_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_feature_id` (`feature_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create families table
CREATE TABLE IF NOT EXISTS `families` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `dynasty_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_dynasty_id` (`dynasty_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create family_members table
CREATE TABLE IF NOT EXISTS `family_members` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `family_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `relationship` varchar(191) NOT NULL DEFAULT 'owner',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_family_id` (`family_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_relationship` (`relationship`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create join_requests table
CREATE TABLE IF NOT EXISTS `join_requests` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `from_user` bigint(20) unsigned NOT NULL,
  `to_user` bigint(20) unsigned NOT NULL,
  `status` smallint(6) NOT NULL DEFAULT 0,
  `relationship` varchar(191) NOT NULL,
  `message` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_from_user` (`from_user`),
  KEY `idx_to_user` (`to_user`),
  KEY `idx_status` (`status`),
  KEY `idx_relationship` (`relationship`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create children_permissions table
CREATE TABLE IF NOT EXISTS `children_permissions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `verified` tinyint(1) NOT NULL DEFAULT 0,
  `BFR` tinyint(1) NOT NULL DEFAULT 0,
  `SF` tinyint(1) NOT NULL DEFAULT 0,
  `W` tinyint(1) NOT NULL DEFAULT 0,
  `JU` tinyint(1) NOT NULL DEFAULT 0,
  `DM` tinyint(1) NOT NULL DEFAULT 0,
  `PIUP` tinyint(1) NOT NULL DEFAULT 0,
  `PITC` tinyint(1) NOT NULL DEFAULT 0,
  `PIC` tinyint(1) NOT NULL DEFAULT 0,
  `ESOO` tinyint(1) NOT NULL DEFAULT 0,
  `COTB` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create dynasty_permissions table (default permissions template)
CREATE TABLE IF NOT EXISTS `dynasty_permissions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `BFR` tinyint(1) NOT NULL DEFAULT 0,
  `SF` tinyint(1) NOT NULL DEFAULT 0,
  `W` tinyint(1) NOT NULL DEFAULT 0,
  `JU` tinyint(1) NOT NULL DEFAULT 0,
  `DM` tinyint(1) NOT NULL DEFAULT 0,
  `PIUP` tinyint(1) NOT NULL DEFAULT 0,
  `PITC` tinyint(1) NOT NULL DEFAULT 0,
  `PIC` tinyint(1) NOT NULL DEFAULT 0,
  `ESOO` tinyint(1) NOT NULL DEFAULT 0,
  `COTB` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create dynasty_prizes table
CREATE TABLE IF NOT EXISTS `dynasty_prizes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `member` varchar(191) NOT NULL,
  `satisfaction` double(8,2) NOT NULL DEFAULT 0.00,
  `introduction_profit_increase` double(8,2) NOT NULL DEFAULT 0.00,
  `accumulated_capital_reserve` double(8,2) NOT NULL DEFAULT 0.00,
  `data_storage` double(8,2) NOT NULL DEFAULT 0.00,
  `psc` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_member` (`member`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create received_prizes table (note: table name is received_prizes, not recieved_prizes)
CREATE TABLE IF NOT EXISTS `received_prizes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `prize_id` bigint(20) unsigned NOT NULL,
  `message` longtext NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_prize_id` (`prize_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create dynasty_messages table (message templates)
CREATE TABLE IF NOT EXISTS `dynasty_messages` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `type` varchar(191) NOT NULL DEFAULT 'invitation',
  `message` text NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default dynasty permissions
INSERT IGNORE INTO `dynasty_permissions` (`id`, `BFR`, `SF`, `W`, `JU`, `DM`, `PIUP`, `PITC`, `PIC`, `ESOO`, `COTB`, `created_at`, `updated_at`)
VALUES (1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, NOW(), NOW());

-- Insert default dynasty message templates
INSERT IGNORE INTO `dynasty_messages` (`type`, `message`, `created_at`, `updated_at`)
VALUES
  ('requester_confirmation_message', 'درخواست شما برای اضافه کردن [relationship] به نام [reciever-name] با کد [reciever-code] در تاریخ [created_at] ثبت شد.', NOW(), NOW()),
  ('reciever_message', 'کاربر [sender-name] با کد [sender-code] درخواست اضافه کردن شما به عنوان [relationship] در تاریخ [created_at] را فرستاده است.', NOW(), NOW()),
  ('requester_accept_message', 'درخواست شما برای اضافه کردن [relationship] به نام [reciever-name] با کد [reciever-code] در تاریخ [created_at] پذیرفته شد.', NOW(), NOW()),
  ('reciever_accept_message', 'شما درخواست [sender-name] با کد [sender-code] برای اضافه شدن به عنوان [relationship] در تاریخ [created_at] را پذیرفتید.', NOW(), NOW()),
  ('requester_reject_message', 'درخواست شما برای اضافه کردن [relationship] به نام [reciever-name] با کد [reciever-code] رد شد.', NOW(), NOW()),
  ('reciever_reject_message', 'شما درخواست [sender-name] با کد [sender-code] برای اضافه شدن به عنوان [relationship] را رد کردید.', NOW(), NOW());

