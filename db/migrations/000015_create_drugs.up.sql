CREATE TABLE IF NOT EXISTS drugs (
    id             VARCHAR(36) PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    aliases        JSON NOT NULL DEFAULT ('[]'),
    spec           VARCHAR(255) NOT NULL,
    default_dosage VARCHAR(255) NOT NULL,
    default_days   INT NOT NULL,
    unit_price     DECIMAL(10,2) NOT NULL,
    stock_quantity INT NOT NULL,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_drugs_name (name),
    KEY idx_drugs_enabled_name (enabled, name),
    CONSTRAINT chk_drugs_default_days_positive CHECK (default_days > 0),
    CONSTRAINT chk_drugs_unit_price_non_negative CHECK (unit_price >= 0),
    CONSTRAINT chk_drugs_stock_quantity_non_negative CHECK (stock_quantity >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO drugs (id, name, aliases, spec, default_dosage, default_days, unit_price, stock_quantity, enabled)
VALUES
    ('drug-buprofen-tablet', '布洛芬片', JSON_ARRAY('布洛芬', '布洛芬缓释胶囊'), '每盒24粒×0.3g', '0.3g，每日2次，餐后服用', 3, 18.50, 200, TRUE),
    ('drug-amoxicillin-capsule', '阿莫西林胶囊', JSON_ARRAY('阿莫西林'), '每盒24粒×0.25g', '0.5g，每日3次，餐后服用', 5, 26.00, 200, TRUE),
    ('drug-acetaminophen-tablet', '对乙酰氨基酚片', JSON_ARRAY('扑热息痛'), '每盒20片×0.5g', '0.5g，每日3次，必要时服用', 3, 12.00, 150, TRUE),
    ('drug-loratadine-tablet', '氯雷他定片', JSON_ARRAY('氯雷他定'), '每盒12片×10mg', '10mg，每日1次', 7, 22.00, 120, TRUE),
    ('drug-ambroxol-tablet', '盐酸氨溴索片', JSON_ARRAY('氨溴索'), '每盒20片×30mg', '30mg，每日3次，餐后服用', 5, 19.80, 120, TRUE)
ON DUPLICATE KEY UPDATE
    aliases = VALUES(aliases),
    spec = VALUES(spec),
    default_dosage = VALUES(default_dosage),
    default_days = VALUES(default_days),
    unit_price = VALUES(unit_price),
    stock_quantity = GREATEST(stock_quantity, VALUES(stock_quantity)),
    enabled = VALUES(enabled);
