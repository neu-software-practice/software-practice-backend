-- Test seed data for NEUHIS Agent
-- Insert sample patients
INSERT INTO patients (id, name, gender, age, phone_masked, id_card_masked, allergies, chronic_diseases, long_term_medications) VALUES
('p001', '张三', 'male', 35, '138****1234', '3101**********1234', '["青霉素"]', '["高血压"]', '["硝苯地平"]'),
('p002', '李四', 'female', 28, '139****5678', '3101**********5678', '[]', '[]', '[]'),
('p003', '王五', 'male', 65, '137****9012', '3101**********9012', '["头孢"]', '["糖尿病", "冠心病"]', '["二甲双胍", "阿司匹林"]');

-- Insert sample visits
INSERT INTO visits (id, patient_id, patient_name, entry_type, status, machine_state, started_at, updated_at, ended_at, timeout_at, paused_at, last_activity_at, ask_round, ask_round_limit, lab_round, lab_round_limit, parent_session_id, terminal_reason, active_card_id, medagent_session_id, timer_paused, summary) VALUES
('v001', 'p001', '张三', 'new', 'completed', 'completed', '2026-06-01 09:00:00', '2026-06-01 09:30:00', '2026-06-01 09:30:00', NULL, NULL, '2026-06-01 09:25:00', 3, 20, 1, 10, NULL, 'completed', NULL, 'ma-sess-001', 0, '{"chiefComplaint": "发热咳嗽", "diagnosis": "急性上呼吸道感染", "treatmentSummary": "布洛芬缓释胶囊"}'),
('v002', 'p002', '李四', 'new', 'chatting', 'chatting', '2026-06-15 14:00:00', '2026-06-15 14:05:00', NULL, NULL, NULL, '2026-06-15 14:05:00', 1, 20, 0, 10, NULL, NULL, NULL, 'ma-sess-002', 0, '{"chiefComplaint": "头痛"}');

-- Insert sample timeline items
INSERT INTO timeline_items (id, session_id, kind, status, content, created_at) VALUES
('t001', 'v001', 'message', 'done', '{"role": "patient", "content": "我发烧咳嗽三天了"}', '2026-06-01 09:00:10'),
('t002', 'v001', 'message', 'done', '{"role": "assistant", "content": "请问您体温多少度？"}', '2026-06-01 09:00:15'),
('t003', 'v001', 'system_event', 'done', '{"eventType": "context_loaded", "title": "上下文加载完成"}', '2026-06-01 09:00:05'),
('t004', 'v001', 'terminal', 'done', '{"reason": "completed", "title": "就诊完成", "description": "诊断结果：急性上呼吸道感染"}', '2026-06-01 09:30:00');
