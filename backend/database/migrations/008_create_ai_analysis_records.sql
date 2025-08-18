-- 创建AI分析记录表
CREATE TABLE IF NOT EXISTS ai_analysis_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    url TEXT NOT NULL,
    analysis_type VARCHAR(50) NOT NULL DEFAULT 'comprehensive',
    engine_type VARCHAR(20) NOT NULL DEFAULT 'dashscope',
    result LONGTEXT,
    status VARCHAR(20) DEFAULT 'completed',
    error_message TEXT,
    process_time INT DEFAULT 0 COMMENT '处理时间（毫秒）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_engine_type (engine_type),
    INDEX idx_status (status),
    INDEX idx_analysis_type (analysis_type),
    CONSTRAINT fk_ai_analysis_user 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI分析记录表'; 