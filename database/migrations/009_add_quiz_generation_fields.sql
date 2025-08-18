-- 练习生成模块 - 为quizzes表添加生成相关字段
-- 创建时间: 2025年1月27日

-- 为quizzes表添加生成来源相关字段
-- 检查并添加字段（如果不存在）

-- 添加generation_source字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND COLUMN_NAME = 'generation_source') = 0,
    'ALTER TABLE quizzes ADD COLUMN generation_source VARCHAR(50) DEFAULT ''manual'' COMMENT ''生成来源: manual/ai_workflow''',
    'SELECT ''Column generation_source already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加workflow_id字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND COLUMN_NAME = 'workflow_id') = 0,
    'ALTER TABLE quizzes ADD COLUMN workflow_id VARCHAR(100) DEFAULT NULL COMMENT ''AI工作流ID''',
    'SELECT ''Column workflow_id already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加generation_params字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND COLUMN_NAME = 'generation_params') = 0,
    'ALTER TABLE quizzes ADD COLUMN generation_params JSON DEFAULT NULL COMMENT ''生成参数''',
    'SELECT ''Column generation_params already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加source_url字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND COLUMN_NAME = 'source_url') = 0,
    'ALTER TABLE quizzes ADD COLUMN source_url TEXT DEFAULT NULL COMMENT ''生成时使用的源URL''',
    'SELECT ''Column source_url already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 为questions表添加来源ID字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'questions' 
     AND COLUMN_NAME = 'source_question_id') = 0,
    'ALTER TABLE questions ADD COLUMN source_question_id INT DEFAULT NULL COMMENT ''来源题目ID(工作流返回的ID)''',
    'SELECT ''Column source_question_id already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加索引优化查询性能（检查是否存在后创建）

-- 创建generation_source索引
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND INDEX_NAME = 'idx_quizzes_generation_source') = 0,
    'CREATE INDEX idx_quizzes_generation_source ON quizzes(generation_source)',
    'SELECT ''Index idx_quizzes_generation_source already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 创建workflow_id索引
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND INDEX_NAME = 'idx_quizzes_workflow_id') = 0,
    'CREATE INDEX idx_quizzes_workflow_id ON quizzes(workflow_id)',
    'SELECT ''Index idx_quizzes_workflow_id already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 创建source_question_id索引
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'questions' 
     AND INDEX_NAME = 'idx_questions_source_question_id') = 0,
    'CREATE INDEX idx_questions_source_question_id ON questions(source_question_id)',
    'SELECT ''Index idx_questions_source_question_id already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 创建复合索引
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'quizzes' 
     AND INDEX_NAME = 'idx_quizzes_course_user') = 0,
    'CREATE INDEX idx_quizzes_course_user ON quizzes(course_id, user_id)',
    'SELECT ''Index idx_quizzes_course_user already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'ai_classroom' 
     AND TABLE_NAME = 'questions' 
     AND INDEX_NAME = 'idx_questions_quiz_order') = 0,
    'CREATE INDEX idx_questions_quiz_order ON questions(quiz_id, order_num)',
    'SELECT ''Index idx_questions_quiz_order already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;