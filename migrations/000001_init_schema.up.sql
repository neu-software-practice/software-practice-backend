-- Initial schema for the HIS outpatient backend (SPEC §6: 15 core tables +
-- 2 financial ledgers authorized by §6 補全). All ids/FKs are BIGINT UNSIGNED to
-- match GORM's uint. Money columns are DECIMAL(8,2). Charset utf8mb4 for Chinese.

CREATE TABLE department (
    id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    dept_code VARCHAR(64)  NOT NULL DEFAULT '',
    dept_name VARCHAR(64)  NOT NULL DEFAULT '',
    dept_type VARCHAR(64)  NOT NULL DEFAULT '',
    delmark   INT          NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_department_dept_type (dept_type),
    KEY idx_department_delmark (delmark)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE regist_level (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    regist_code  VARCHAR(64)  NOT NULL DEFAULT '',
    regist_name  VARCHAR(64)  NOT NULL DEFAULT '',
    regist_fee   DECIMAL(8,2) NOT NULL DEFAULT 0.00,
    regist_quota INT          NOT NULL DEFAULT 0,
    sequence_no  INT          NOT NULL DEFAULT 0,
    delmark      INT          NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_regist_level_delmark (delmark)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE settle_category (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    settle_code VARCHAR(64) NOT NULL DEFAULT '',
    settle_name VARCHAR(64) NOT NULL DEFAULT '',
    sequence_no INT         NOT NULL DEFAULT 0,
    delmark     INT         NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_settle_category_delmark (delmark)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE scheduling (
    id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    rule_name VARCHAR(64) NOT NULL DEFAULT '',
    week_rule VARCHAR(32) NOT NULL DEFAULT '',
    delmark   INT         NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_scheduling_delmark (delmark)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE employee (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username        VARCHAR(64)  NOT NULL DEFAULT '',
    password        VARCHAR(128) NOT NULL DEFAULT '',
    realname        VARCHAR(64)  NOT NULL DEFAULT '',
    deptment_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    regist_level_id BIGINT UNSIGNED NULL,
    scheduling_id   BIGINT UNSIGNED NULL,
    delmark         INT          NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uniq_employee_username (username),
    KEY idx_employee_deptment_id (deptment_id),
    KEY idx_employee_delmark (delmark)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE register (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    case_number        VARCHAR(64)  NOT NULL DEFAULT '',
    real_name          VARCHAR(64)  NOT NULL DEFAULT '',
    gender             VARCHAR(6)   NOT NULL DEFAULT '',
    card_number        VARCHAR(18)  NOT NULL DEFAULT '',
    birthdate          DATE         NULL,
    age                INT          NOT NULL DEFAULT 0,
    age_type           VARCHAR(6)   NOT NULL DEFAULT '',
    home_address       VARCHAR(128) NOT NULL DEFAULT '',
    visit_date         DATETIME     NULL,
    noon               VARCHAR(6)   NOT NULL DEFAULT '',
    deptment_id        BIGINT UNSIGNED NOT NULL DEFAULT 0,
    employee_id        BIGINT UNSIGNED NOT NULL DEFAULT 0,
    regist_level_id    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    settle_category_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    is_book            VARCHAR(2)   NOT NULL DEFAULT '',
    regist_method      VARCHAR(10)  NOT NULL DEFAULT '',
    regist_money       DECIMAL(8,2) NOT NULL DEFAULT 0.00,
    visit_state        INT          NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_register_case_number (case_number),
    KEY idx_register_real_name (real_name),
    KEY idx_register_deptment_id (deptment_id),
    KEY idx_register_employee_id (employee_id),
    KEY idx_register_visit_state (visit_state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE medical_technology (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tech_code   VARCHAR(64)  NOT NULL DEFAULT '',
    tech_name   VARCHAR(64)  NOT NULL DEFAULT '',
    tech_format VARCHAR(64)  NOT NULL DEFAULT '',
    tech_price  DECIMAL(8,2) NOT NULL DEFAULT 0.00,
    tech_type   VARCHAR(64)  NOT NULL DEFAULT '',
    price_type  VARCHAR(64)  NOT NULL DEFAULT '',
    deptment_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_medical_technology_tech_code (tech_code),
    KEY idx_medical_technology_tech_name (tech_name),
    KEY idx_medical_technology_tech_type (tech_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE check_request (
    id                     BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    register_id            BIGINT UNSIGNED NOT NULL DEFAULT 0,
    medical_technology_id  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    check_info             VARCHAR(512) NOT NULL DEFAULT '',
    check_position         VARCHAR(255) NOT NULL DEFAULT '',
    creation_time          DATETIME     NULL,
    check_employee_id      BIGINT UNSIGNED NULL,
    inputcheck_employee_id BIGINT UNSIGNED NULL,
    check_time             DATETIME     NULL,
    check_result           VARCHAR(512) NOT NULL DEFAULT '',
    check_state            VARCHAR(64)  NOT NULL DEFAULT '',
    check_remark           VARCHAR(512) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_check_request_register_id (register_id),
    KEY idx_check_request_check_state (check_state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE inspection_request (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    register_id                 BIGINT UNSIGNED NOT NULL DEFAULT 0,
    medical_technology_id       BIGINT UNSIGNED NOT NULL DEFAULT 0,
    inspection_info             VARCHAR(512) NOT NULL DEFAULT '',
    inspection_position         VARCHAR(255) NOT NULL DEFAULT '',
    creation_time               DATETIME     NULL,
    inspection_employee_id      BIGINT UNSIGNED NULL,
    inputinspection_employee_id BIGINT UNSIGNED NULL,
    inspection_time             DATETIME     NULL,
    inspection_result           VARCHAR(512) NOT NULL DEFAULT '',
    inspection_state            VARCHAR(64)  NOT NULL DEFAULT '',
    inspection_remark           VARCHAR(512) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_inspection_request_register_id (register_id),
    KEY idx_inspection_request_inspection_state (inspection_state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE disposal_request (
    id                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    register_id              BIGINT UNSIGNED NOT NULL DEFAULT 0,
    medical_technology_id    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    disposal_info            VARCHAR(512) NOT NULL DEFAULT '',
    disposal_position        VARCHAR(255) NOT NULL DEFAULT '',
    creation_time            DATETIME     NULL,
    disposal_employee_id     BIGINT UNSIGNED NULL,
    inputdisposal_employee_id BIGINT UNSIGNED NULL,
    disposal_time            DATETIME     NULL,
    disposal_result          VARCHAR(512) NOT NULL DEFAULT '',
    disposal_state           VARCHAR(64)  NOT NULL DEFAULT '',
    disposal_remark          VARCHAR(512) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_disposal_request_register_id (register_id),
    KEY idx_disposal_request_disposal_state (disposal_state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE medical_record (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    register_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    readme        VARCHAR(512) NOT NULL DEFAULT '',
    present       VARCHAR(512) NOT NULL DEFAULT '',
    present_treat VARCHAR(512) NOT NULL DEFAULT '',
    history       VARCHAR(512) NOT NULL DEFAULT '',
    allergy       VARCHAR(512) NOT NULL DEFAULT '',
    physique      VARCHAR(512) NOT NULL DEFAULT '',
    proposal      VARCHAR(512) NOT NULL DEFAULT '',
    careful       VARCHAR(512) NOT NULL DEFAULT '',
    diagnosis     VARCHAR(512) NOT NULL DEFAULT '',
    cure          VARCHAR(512) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    UNIQUE KEY uniq_medical_record_register_id (register_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE disease (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    disease_code     VARCHAR(64)  NOT NULL DEFAULT '',
    disease_name     VARCHAR(255) NOT NULL DEFAULT '',
    disease_icd      VARCHAR(64)  NOT NULL DEFAULT '',
    disease_category VARCHAR(64)  NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_disease_disease_code (disease_code),
    KEY idx_disease_disease_name (disease_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE medical_record_disease (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    medical_record_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    disease_id        BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uniq_record_disease (medical_record_id, disease_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE drug_info (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    drug_code     VARCHAR(255) NOT NULL DEFAULT '',
    drug_name     VARCHAR(255) NOT NULL DEFAULT '',
    drug_format   VARCHAR(255) NOT NULL DEFAULT '',
    drug_unit     VARCHAR(16)  NOT NULL DEFAULT '',
    manufacturer  VARCHAR(255) NOT NULL DEFAULT '',
    drug_dosage   VARCHAR(64)  NOT NULL DEFAULT '',
    drug_type     VARCHAR(64)  NOT NULL DEFAULT '',
    drug_price    DECIMAL(8,2) NOT NULL DEFAULT 0.00,
    drug_stock    INT          NOT NULL DEFAULT 0,
    mnemonic_code VARCHAR(255) NOT NULL DEFAULT '',
    creation_date DATE         NULL,
    delmark       INT          NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_drug_info_drug_code (drug_code),
    KEY idx_drug_info_drug_name (drug_name),
    KEY idx_drug_info_mnemonic_code (mnemonic_code),
    KEY idx_drug_info_delmark (delmark)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE prescription (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    register_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    drug_id       BIGINT UNSIGNED NOT NULL DEFAULT 0,
    drug_usage    VARCHAR(255) NOT NULL DEFAULT '',
    drug_number   INT          NOT NULL DEFAULT 0,
    creation_time DATETIME     NULL,
    drug_state    VARCHAR(64)  NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_prescription_register_id (register_id),
    KEY idx_prescription_drug_state (drug_state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE charge_record (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    register_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    item_type   VARCHAR(32)  NOT NULL DEFAULT '',
    item_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    item_name   VARCHAR(128) NOT NULL DEFAULT '',
    amount      DECIMAL(8,2) NOT NULL DEFAULT 0.00,
    action      VARCHAR(16)  NOT NULL DEFAULT '',
    operator_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at  DATETIME     NULL,
    PRIMARY KEY (id),
    KEY idx_charge_record_register_id (register_id),
    KEY idx_charge_record_item_type (item_type),
    KEY idx_charge_record_action (action),
    KEY idx_charge_record_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE drug_transaction (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    prescription_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    register_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    drug_id         BIGINT UNSIGNED NOT NULL DEFAULT 0,
    drug_name       VARCHAR(255) NOT NULL DEFAULT '',
    quantity        INT          NOT NULL DEFAULT 0,
    action          VARCHAR(16)  NOT NULL DEFAULT '',
    operator_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at      DATETIME     NULL,
    PRIMARY KEY (id),
    KEY idx_drug_transaction_prescription_id (prescription_id),
    KEY idx_drug_transaction_register_id (register_id),
    KEY idx_drug_transaction_action (action),
    KEY idx_drug_transaction_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
