-- 练习生成模块 - 创建练习生成记录表
-- 创建时间: 2025年1月27日

CREATE TABLE IF NOT EXISTS exercise_generation_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL COMMENT '生成用户ID',
    course_id BIGINT UNSIGNED NOT NULL COMMENT '课程ID',
    quiz_id BIGINT UNSIGNED NULL COMMENT '生成的测验ID',
    workflow_id VARCHAR(100) NOT NULL COMMENT '使用的工作流ID',
    source_url TEXT NOT NULL COMMENT '课程源URL',
    generation_status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending' COMMENT '生成状态',
    request_data JSON NULL COMMENT '请求参数数据',
    response_data JSON NULL COMMENT '工作流响应数据',
    error_message TEXT NULL COMMENT '错误信息',
    processing_duration INT DEFAULT 0 COMMENT '处理时长(秒)',
    total_questions INT DEFAULT 0 COMMENT '生成题目总数',
    difficulty_distribution JSON NULL COMMENT '难度分布统计',
    created_at DATETIME(3) DEFAULT NULL COMMENT '创建时间',
    updated_at DATETIME(3) DEFAULT NULL COMMENT '更新时间',
    
    -- 索引设计
    INDEX idx_user_id (user_id),
    INDEX idx_course_id (course_id),
    INDEX idx_quiz_id (quiz_id),
    INDEX idx_generation_status (generation_status),
    INDEX idx_workflow_id (workflow_id),
    INDEX idx_created_at (created_at),
    INDEX idx_user_course (user_id, course_id),
    
    -- 外键约束
    CONSTRAINT fk_exercise_generation_user 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_exercise_generation_course 
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    CONSTRAINT fk_exercise_generation_quiz 
        FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE SET NULL
        
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci 
COMMENT='练习生成记录表 - 记录AI工作流生成练习的详细信息';