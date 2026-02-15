-- Grant can_use_bot to all existing dark/admin users (mirrors config.grantCanUseBotToDarkAndAdmin).
-- Run after permissions are initialized (permissions.id for name = 'can_use_bot' must exist).
-- Idempotent: skips users who already have the permission (including soft-deleted user_permissions).

-- Shard 0: users from users_shard_0 with account_type IN ('dark', 'admin')
INSERT INTO user_permissions (user_id, permission_id, shard_number, created_at, updated_at)
SELECT u.id, p.id, 0, NOW(), NOW()
FROM users_shard_0 u
CROSS JOIN (SELECT id FROM permissions WHERE name = 'can_use_bot' AND deleted_at IS NULL LIMIT 1) p
WHERE u.account_type IN ('dark', 'admin')
  AND u.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM user_permissions up
    WHERE up.user_id = u.id AND up.permission_id = p.id AND up.deleted_at IS NULL
  );

-- Shard 1: users from users_shard_1 with account_type IN ('dark', 'admin')
INSERT INTO user_permissions (user_id, permission_id, shard_number, created_at, updated_at)
SELECT u.id, p.id, 1, NOW(), NOW()
FROM users_shard_1 u
CROSS JOIN (SELECT id FROM permissions WHERE name = 'can_use_bot' AND deleted_at IS NULL LIMIT 1) p
WHERE u.account_type IN ('dark', 'admin')
  AND u.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM user_permissions up
    WHERE up.user_id = u.id AND up.permission_id = p.id AND up.deleted_at IS NULL
  );
