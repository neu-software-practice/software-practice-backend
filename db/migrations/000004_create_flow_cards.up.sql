CREATE TABLE IF NOT EXISTS flow_cards (
    id          VARCHAR(36)  PRIMARY KEY,
    session_id  VARCHAR(36)  NOT NULL,
    kind        VARCHAR(30)  NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending',
    blocking    TINYINT(1)   NOT NULL DEFAULT 0,
    title       VARCHAR(500) NOT NULL DEFAULT '',
    content     JSON         NOT NULL,
    lock_reason VARCHAR(500) DEFAULT '',
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    handled_at  TIMESTAMP    NULL,
    FOREIGN KEY (session_id) REFERENCES visits(id) ON DELETE CASCADE,
    INDEX idx_flow_cards_session_id (session_id),
    INDEX idx_flow_cards_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
