-- Test data fixtures for integration and golden tests
-- This file contains deterministic test data for reproducible tests

-- Clean up existing test data
DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%');
DELETE FROM buy_feature_requests WHERE sender_id IN (SELECT id FROM users WHERE username LIKE 'test_%');
DELETE FROM transactions WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%');
DELETE FROM feature_properties WHERE feature_id IN (SELECT id FROM features WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%'));
DELETE FROM geometries WHERE feature_id IN (SELECT id FROM features WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%'));
DELETE FROM features WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%');
DELETE FROM wallets WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%');
DELETE FROM personal_access_tokens WHERE tokenable_id IN (SELECT id FROM users WHERE username LIKE 'test_%');
DELETE FROM users WHERE username LIKE 'test_%';

-- Test users
INSERT INTO users (id, username, email, password, email_verified_at, created_at, updated_at, last_seen) VALUES
(9001, 'test_golden_user', 'test_golden@example.com', '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', NOW(), '2024-01-01 10:00:00', '2024-01-01 10:00:00', '2024-01-15 14:30:00'),
(9002, 'test_buyer_user', 'test_buyer@example.com', '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', NOW(), '2024-01-02 10:00:00', '2024-01-02 10:00:00', '2024-01-15 15:00:00'),
(9003, 'test_seller_user', 'test_seller@example.com', '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', NOW(), '2024-01-03 10:00:00', '2024-01-03 10:00:00', '2024-01-15 16:00:00');

-- Test tokens
INSERT INTO personal_access_tokens (id, tokenable_type, tokenable_id, name, token, abilities, expires_at, created_at, updated_at) VALUES
(9001, 'App\\Models\\User', 9001, 'test-device', 'test_token_9001', '["*"]', NULL, NOW(), NOW()),
(9002, 'App\\Models\\User', 9002, 'test-device', 'test_token_9002', '["*"]', NULL, NOW(), NOW()),
(9003, 'App\\Models\\User', 9003, 'test-device', 'test_token_9003', '["*"]', NULL, NOW(), NOW());

-- Test wallets
INSERT INTO wallets (id, user_id, psc, rgb, created_at, updated_at) VALUES
(9001, 9001, '10000.0000000000', '500.0000000000', '2024-01-01 10:00:00', '2024-01-15 14:30:00'),
(9002, 9002, '25000.5000000000', '1200.2500000000', '2024-01-02 10:00:00', '2024-01-15 15:00:00'),
(9003, 9003, '5000.0000000000', '100.0000000000', '2024-01-03 10:00:00', '2024-01-15 16:00:00');

-- Test features
INSERT INTO features (id, user_id, status, created_at, updated_at) VALUES
('F-TEST-001', 9003, 'active', '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
('F-TEST-002', 9003, 'active', '2024-01-06 10:00:00', '2024-01-06 10:00:00'),
('F-TEST-003', 9001, 'active', '2024-01-07 10:00:00', '2024-01-07 10:00:00');

-- Test feature properties
INSERT INTO feature_properties (id, feature_id, price_psc, price_irr, for_sale, created_at, updated_at) VALUES
('PROP-F-TEST-001', 'F-TEST-001', '1000', '50000000', 1, '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
('PROP-F-TEST-002', 'F-TEST-002', '2500', '125000000', 1, '2024-01-06 10:00:00', '2024-01-06 10:00:00'),
('PROP-F-TEST-003', 'F-TEST-003', '1500', '75000000', 0, '2024-01-07 10:00:00', '2024-01-07 10:00:00');

-- Test geometries
INSERT INTO geometries (id, feature_id, type, created_at, updated_at) VALUES
(9001, 'F-TEST-001', 'Polygon', '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
(9002, 'F-TEST-002', 'Polygon', '2024-01-06 10:00:00', '2024-01-06 10:00:00'),
(9003, 'F-TEST-003', 'Polygon', '2024-01-07 10:00:00', '2024-01-07 10:00:00');

-- Test coordinates
INSERT INTO coordinates (id, geometry_id, lat, lng, sequence, created_at, updated_at) VALUES
(90001, 9001, 35.6892, 51.3890, 0, '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
(90002, 9001, 35.6892, 51.3900, 1, '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
(90003, 9001, 35.6902, 51.3900, 2, '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
(90004, 9001, 35.6902, 51.3890, 3, '2024-01-05 10:00:00', '2024-01-05 10:00:00'),
(90005, 9002, 35.7000, 51.4000, 0, '2024-01-06 10:00:00', '2024-01-06 10:00:00'),
(90006, 9002, 35.7000, 51.4010, 1, '2024-01-06 10:00:00', '2024-01-06 10:00:00'),
(90007, 9002, 35.7010, 51.4010, 2, '2024-01-06 10:00:00', '2024-01-06 10:00:00'),
(90008, 9002, 35.7010, 51.4000, 3, '2024-01-06 10:00:00', '2024-01-06 10:00:00');

-- Test transactions
INSERT INTO transactions (id, user_id, amount, type, payable_type, payable_id, status, created_at, updated_at) VALUES
('TX-TEST-001', 9001, '1000.0000000000', 'deposit', 'Order', '1', 'completed', '2024-01-10 10:00:00', '2024-01-10 10:00:00'),
('TX-TEST-002', 9001, '500.0000000000', 'purchase', 'Feature', 'F-TEST-003', 'completed', '2024-01-11 10:00:00', '2024-01-11 10:00:00'),
('TX-TEST-003', 9002, '2000.0000000000', 'deposit', 'Order', '2', 'completed', '2024-01-12 10:00:00', '2024-01-12 10:00:00');

-- Test notifications
INSERT INTO notifications (id, user_id, type, title, message, read_at, created_at, updated_at) VALUES
(UUID(), 9001, 'system', 'خوش آمدید', 'به متارجی‌بی خوش آمدید', NULL, '2024-01-01 10:05:00', '2024-01-01 10:05:00'),
(UUID(), 9001, 'transaction', 'تراکنش موفق', 'تراکنش شما با موفقیت انجام شد', '2024-01-10 10:05:00', '2024-01-10 10:01:00', '2024-01-10 10:05:00'),
(UUID(), 9002, 'system', 'خوش آمدید', 'به متارجی‌بی خوش آمدید', NULL, '2024-01-02 10:05:00', '2024-01-02 10:05:00');

-- Test levels
INSERT INTO levels (id, name, required_score, created_at, updated_at) VALUES
(1, 'سطح 1', 0, '2024-01-01 00:00:00', '2024-01-01 00:00:00'),
(2, 'سطح 2', 100, '2024-01-01 00:00:00', '2024-01-01 00:00:00'),
(3, 'سطح 3', 500, '2024-01-01 00:00:00', '2024-01-01 00:00:00');

-- Test user levels
INSERT INTO level_user (user_id, level_id, score, created_at, updated_at) VALUES
(9001, 1, 50, '2024-01-01 10:00:00', '2024-01-15 14:30:00'),
(9002, 2, 150, '2024-01-02 10:00:00', '2024-01-15 15:00:00'),
(9003, 1, 25, '2024-01-03 10:00:00', '2024-01-15 16:00:00');

-- Test dynasties
INSERT INTO dynasties (id, name, feature_id, founder_id, created_at, updated_at) VALUES
(9001, 'سلسله تست', 'F-TEST-001', 9003, '2024-01-08 10:00:00', '2024-01-08 10:00:00');

-- Test families
INSERT INTO families (id, dynasty_id, name, created_at, updated_at) VALUES
(9001, 9001, 'خانواده اصلی', '2024-01-08 10:00:00', '2024-01-08 10:00:00');

-- Test family members
INSERT INTO family_members (id, family_id, user_id, role, relationship, created_at, updated_at) VALUES
(9001, 9001, 9003, 'founder', 'self', '2024-01-08 10:00:00', '2024-01-08 10:00:00');

COMMIT;

