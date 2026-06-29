ALTER TABLE patients ADD COLUMN medical_history JSON DEFAULT ('[]') AFTER long_term_medications;
