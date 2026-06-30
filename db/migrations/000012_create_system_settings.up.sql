CREATE TABLE IF NOT EXISTS system_settings (
    id INT PRIMARY KEY DEFAULT 1,
    site_name VARCHAR(255) NOT NULL DEFAULT 'NEUHIS Agent',
    max_concurrent_sessions INT NOT NULL DEFAULT 3,
    session_timeout_minutes INT NOT NULL DEFAULT 30,
    enable_registration BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed default settings
INSERT INTO system_settings (id, site_name, max_concurrent_sessions, session_timeout_minutes, enable_registration)
VALUES (1, 'NEUHIS Agent', 3, 30, TRUE)
ON DUPLICATE KEY UPDATE id=id;
