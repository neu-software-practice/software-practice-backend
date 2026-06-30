CREATE TABLE IF NOT EXISTS addresses (
    id          VARCHAR(36)  PRIMARY KEY,
    patient_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(20)  NOT NULL,
    phone       VARCHAR(11)  NOT NULL,
    province    VARCHAR(50)  NOT NULL,
    city        VARCHAR(50)  NOT NULL,
    district    VARCHAR(50)  NOT NULL,
    detail      VARCHAR(200) NOT NULL,
    is_default  TINYINT(1)   NOT NULL DEFAULT 0,
    tag         VARCHAR(10)  NOT NULL DEFAULT '',
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE CASCADE,
    INDEX idx_addresses_patient_id (patient_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
