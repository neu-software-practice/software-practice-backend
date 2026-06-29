CREATE TABLE IF NOT EXISTS patients (
    id          VARCHAR(36)  PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    gender      VARCHAR(10)  NOT NULL DEFAULT 'unknown',
    age         INT          NOT NULL DEFAULT 0,
    phone_masked    VARCHAR(20)  DEFAULT '',
    id_card_masked  VARCHAR(20)  DEFAULT '',
    allergies          JSON DEFAULT ('[]'),
    chronic_diseases   JSON DEFAULT ('[]'),
    long_term_medications JSON DEFAULT ('[]'),
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
