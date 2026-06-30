CREATE TABLE IF NOT EXISTS admin_users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'admin',
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed default super_admin account: admin / admin123
INSERT INTO admin_users (id, username, password_hash, role, display_name, created_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin',
    '$2a$12$UyrlMuL8F/xUROa2cDEwUuNNLL9D9QGHPQb8qeO.nIGpOLq5j3h4a',
    'super_admin',
    '系统管理员',
    NOW()
);
