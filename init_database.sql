-- AI课堂数据库初始化脚本
-- 创建时间: 2024-07-17

-- 创建数据库
CREATE DATABASE IF NOT EXISTS ai_classroom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE ai_classroom;

-- 1. 用户表 (users)
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    openid VARCHAR(100) UNIQUE NOT NULL COMMENT '微信用户唯一标识',
    nickname VARCHAR(100) COMMENT '用户昵称',
    avatar_url VARCHAR(500) COMMENT '头像URL',
    phone VARCHAR(20) COMMENT '手机号',
    email VARCHAR(100) COMMENT '邮箱',
    vip_level INT DEFAULT 0 COMMENT 'VIP等级：0-普通用户，1-月度会员，2-年度会员',
    vip_expired_at DATETIME COMMENT 'VIP过期时间',
    credits INT DEFAULT 10 COMMENT '用户积分/次数',
    total_courses_created INT DEFAULT 0 COMMENT '创建课件总数',
    total_study_time INT DEFAULT 0 COMMENT '总学习时长（秒）',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 2. 课件表 (courses)
CREATE TABLE courses (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '创建用户ID',
    title VARCHAR(200) NOT NULL COMMENT '课件标题',
    description TEXT COMMENT '课件描述',
    category VARCHAR(50) DEFAULT 'general' COMMENT '课件分类',
    tags JSON COMMENT '课件标签',
    source_type ENUM('url', 'document', 'text') NOT NULL COMMENT '来源类型',
    source_content TEXT NOT NULL COMMENT '原始内容',
    source_url VARCHAR(1000) COMMENT '原始URL',
    file_path VARCHAR(500) COMMENT '上传文件路径',
    file_size BIGINT COMMENT '文件大小（字节）',
    thumbnail_url VARCHAR(500) COMMENT '缩略图URL',
    status ENUM('generating', 'completed', 'failed') DEFAULT 'generating' COMMENT '生成状态',
    error_message TEXT COMMENT '错误信息',
    slides_count INT DEFAULT 0 COMMENT '幻灯片数量',
    duration INT DEFAULT 0 COMMENT '课件总时长（秒）',
    view_count INT DEFAULT 0 COMMENT '观看次数',
    like_count INT DEFAULT 0 COMMENT '点赞数',
    share_count INT DEFAULT 0 COMMENT '分享次数',
    is_public BOOLEAN DEFAULT FALSE COMMENT '是否公开',
    voice_type VARCHAR(50) DEFAULT 'xiaoyun' COMMENT '语音类型',
    generation_params JSON COMMENT '生成参数',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 3. 幻灯片表 (slides)
CREATE TABLE slides (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    course_id BIGINT NOT NULL COMMENT '课件ID',
    slide_number INT NOT NULL COMMENT '幻灯片序号',
    title VARCHAR(300) COMMENT '幻灯片标题',
    content TEXT NOT NULL COMMENT '幻灯片内容',
    speaker_notes TEXT COMMENT '演讲备注',
    image_url VARCHAR(500) COMMENT '幻灯片图片URL',
    audio_url VARCHAR(500) COMMENT '语音文件URL',
    duration INT COMMENT '音频时长（秒）',
    layout_type VARCHAR(50) DEFAULT 'content' COMMENT '布局类型：title/content/summary',
    transition VARCHAR(50) DEFAULT 'fade' COMMENT '转场效果',
    background_color VARCHAR(20) COMMENT '背景颜色',
    text_color VARCHAR(20) COMMENT '文字颜色',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    UNIQUE KEY uk_course_slide (course_id, slide_number)
);

-- 4. 练习/测验表 (quizzes)
CREATE TABLE quizzes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    course_id BIGINT NOT NULL COMMENT '课件ID',
    title VARCHAR(200) DEFAULT '课后练习' COMMENT '练习标题',
    description TEXT COMMENT '练习描述',
    total_questions INT DEFAULT 0 COMMENT '题目总数',
    total_points INT DEFAULT 0 COMMENT '总分',
    time_limit INT DEFAULT 0 COMMENT '时间限制（分钟），0表示无限制',
    pass_score INT DEFAULT 60 COMMENT '及格分数',
    attempt_limit INT DEFAULT 3 COMMENT '答题次数限制',
    status ENUM('active', 'inactive', 'archived') DEFAULT 'active' COMMENT '状态',
    attempt_count INT DEFAULT 0 COMMENT '答题总次数',
    pass_count INT DEFAULT 0 COMMENT '通过人数',
    average_score DECIMAL(5,2) DEFAULT 0 COMMENT '平均分',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

-- 5. 题目表 (questions)
CREATE TABLE questions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    quiz_id BIGINT NOT NULL COMMENT '练习ID',
    course_id BIGINT NOT NULL COMMENT '课件ID',
    slide_id BIGINT COMMENT '关联幻灯片ID',
    type ENUM('single_choice', 'multiple_choice', 'true_false', 'fill_blank') NOT NULL COMMENT '题目类型',
    question TEXT NOT NULL COMMENT '题目内容',
    options JSON COMMENT '选择题选项',
    correct_answer TEXT NOT NULL COMMENT '正确答案',
    explanation TEXT COMMENT '答案解释',
    difficulty ENUM('easy', 'normal', 'hard') DEFAULT 'normal' COMMENT '难度等级',
    points INT DEFAULT 1 COMMENT '分值',
    order_num INT DEFAULT 0 COMMENT '题目顺序',
    keywords JSON COMMENT '关键词',
    answer_count INT DEFAULT 0 COMMENT '回答次数',
    correct_count INT DEFAULT 0 COMMENT '正确次数',
    correct_rate DECIMAL(5,2) DEFAULT 0 COMMENT '正确率',
    avg_duration DECIMAL(8,2) DEFAULT 0 COMMENT '平均答题时长',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    FOREIGN KEY (slide_id) REFERENCES slides(id) ON DELETE SET NULL
);

-- 6. 用户答题记录表 (quiz_attempts)
CREATE TABLE quiz_attempts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    quiz_id BIGINT NOT NULL COMMENT '练习ID',
    course_id BIGINT NOT NULL COMMENT '课件ID',
    start_time DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '开始时间',
    end_time DATETIME COMMENT '结束时间',
    duration INT DEFAULT 0 COMMENT '答题时长（秒）',
    total_score INT DEFAULT 0 COMMENT '总分',
    user_score INT DEFAULT 0 COMMENT '用户得分',
    percentage DECIMAL(5,2) DEFAULT 0 COMMENT '得分率',
    is_passed BOOLEAN DEFAULT FALSE COMMENT '是否通过',
    is_completed BOOLEAN DEFAULT FALSE COMMENT '是否完成',
    correct_count INT DEFAULT 0 COMMENT '正确题数',
    wrong_count INT DEFAULT 0 COMMENT '错误题数',
    skipped_count INT DEFAULT 0 COMMENT '跳过题数',
    submitted_at DATETIME COMMENT '提交时间',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

-- 7. 用户答案表 (user_answers)
CREATE TABLE user_answers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    question_id BIGINT NOT NULL COMMENT '题目ID',
    attempt_id BIGINT NOT NULL COMMENT '答题记录ID',
    user_answer TEXT COMMENT '用户答案',
    is_correct BOOLEAN DEFAULT FALSE COMMENT '是否正确',
    points INT DEFAULT 0 COMMENT '得分',
    duration INT DEFAULT 0 COMMENT '答题时长（秒）',
    answered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
    FOREIGN KEY (attempt_id) REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_question_attempt (user_id, question_id, attempt_id)
);

-- 8. 学习记录表 (learning_records)
CREATE TABLE learning_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    course_id BIGINT NOT NULL COMMENT '课件ID',
    current_slide INT DEFAULT 1 COMMENT '当前学习到的幻灯片',
    total_slides INT DEFAULT 0 COMMENT '总幻灯片数',
    progress DECIMAL(5,2) DEFAULT 0.00 COMMENT '学习进度百分比',
    study_duration INT DEFAULT 0 COMMENT '学习时长（秒）',
    complete_rate DECIMAL(5,2) DEFAULT 0.00 COMMENT '完成百分比',
    last_position INT DEFAULT 0 COMMENT '最后播放位置',
    quiz_best_score DECIMAL(5,2) COMMENT '最佳练习得分',
    quiz_attempts INT DEFAULT 0 COMMENT '练习尝试次数',
    is_completed BOOLEAN DEFAULT FALSE COMMENT '是否完成学习',
    is_bookmarked BOOLEAN DEFAULT FALSE COMMENT '是否收藏',
    status ENUM('learning', 'completed', 'paused') DEFAULT 'learning' COMMENT '学习状态',
    start_time DATETIME COMMENT '开始学习时间',
    end_time DATETIME COMMENT '完成学习时间',
    last_study_time DATETIME COMMENT '最后学习时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_course (user_id, course_id)
);

-- 9. 分享记录表 (share_records)
CREATE TABLE share_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '分享用户ID',
    course_id BIGINT NOT NULL COMMENT '课件ID',
    share_code VARCHAR(32) UNIQUE NOT NULL COMMENT '分享码',
    title VARCHAR(200) COMMENT '分享标题',
    description VARCHAR(500) COMMENT '分享描述',
    cover_image VARCHAR(500) COMMENT '分享封面图',
    view_count INT DEFAULT 0 COMMENT '查看次数',
    like_count INT DEFAULT 0 COMMENT '点赞次数',
    download_count INT DEFAULT 0 COMMENT '下载次数',
    allow_download BOOLEAN DEFAULT TRUE COMMENT '是否允许下载',
    expires_at DATETIME COMMENT '过期时间',
    is_active BOOLEAN DEFAULT TRUE COMMENT '是否有效',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

-- 10. 分享点赞表 (share_likes)
CREATE TABLE share_likes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    share_id BIGINT NOT NULL COMMENT '分享记录ID',
    user_id BIGINT NOT NULL COMMENT '点赞用户ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (share_id) REFERENCES share_records(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uk_share_user (share_id, user_id)
);

-- 11. 使用记录表 (usage_records)
CREATE TABLE usage_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    type ENUM('course_generation', 'tts_generation', 'quiz_generation') NOT NULL COMMENT '使用类型',
    resource_id BIGINT COMMENT '资源ID（课件ID等）',
    consumed_credits INT DEFAULT 1 COMMENT '消耗积分',
    ip_address VARCHAR(45) COMMENT 'IP地址',
    user_agent VARCHAR(1000) COMMENT '用户代理',
    consumed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 12. 购买记录表 (purchase_records)
CREATE TABLE purchase_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    order_no VARCHAR(64) UNIQUE NOT NULL COMMENT '订单号',
    plan_id INT NOT NULL COMMENT '套餐ID',
    plan_name VARCHAR(100) NOT NULL COMMENT '套餐名称',
    amount INT NOT NULL COMMENT '金额（分）',
    payment_method ENUM('wechat', 'alipay') DEFAULT 'wechat' COMMENT '支付方式',
    payment_id VARCHAR(100) COMMENT '第三方支付ID',
    status ENUM('pending', 'completed', 'failed', 'refunded') DEFAULT 'pending' COMMENT '支付状态',
    paid_at DATETIME COMMENT '支付时间',
    refunded_at DATETIME COMMENT '退款时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 13. 音频元数据表 (audio_metadata)
CREATE TABLE audio_metadata (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    course_id BIGINT NOT NULL COMMENT '课件ID',
    slide_id BIGINT COMMENT '幻灯片ID',
    file_name VARCHAR(255) NOT NULL COMMENT '文件名',
    file_path VARCHAR(500) NOT NULL COMMENT '文件路径',
    file_size BIGINT NOT NULL COMMENT '文件大小（字节）',
    duration INT NOT NULL COMMENT '时长（秒）',
    format VARCHAR(20) DEFAULT 'mp3' COMMENT '音频格式',
    sample_rate INT DEFAULT 16000 COMMENT '采样率',
    bit_rate INT DEFAULT 128 COMMENT '比特率',
    url VARCHAR(500) COMMENT '访问URL',
    cdn_url VARCHAR(500) COMMENT 'CDN URL',
    status ENUM('processing', 'completed', 'failed') DEFAULT 'processing' COMMENT '处理状态',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    FOREIGN KEY (slide_id) REFERENCES slides(id) ON DELETE CASCADE
);

-- 14. 系统配置表 (system_configs)
CREATE TABLE system_configs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    config_key VARCHAR(100) UNIQUE NOT NULL COMMENT '配置键',
    config_value TEXT COMMENT '配置值',
    config_type ENUM('string', 'number', 'boolean', 'json') DEFAULT 'string' COMMENT '配置类型',
    description VARCHAR(500) COMMENT '配置描述',
    is_active BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 15. 操作日志表 (operation_logs)
CREATE TABLE operation_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT COMMENT '用户ID',
    operation_type VARCHAR(50) NOT NULL COMMENT '操作类型',
    operation_desc VARCHAR(500) COMMENT '操作描述',
    resource_type VARCHAR(50) COMMENT '资源类型',
    resource_id BIGINT COMMENT '资源ID',
    request_data JSON COMMENT '请求数据',
    response_data JSON COMMENT '响应数据',
    ip_address VARCHAR(45) COMMENT 'IP地址',
    user_agent VARCHAR(1000) COMMENT '用户代理',
    execution_time INT COMMENT '执行时间（毫秒）',
    status ENUM('success', 'failed') DEFAULT 'success' COMMENT '执行状态',
    error_message TEXT COMMENT '错误信息',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- =============================================
-- 插入样本数据
-- =============================================

-- 1. 插入系统配置数据
INSERT INTO system_configs (config_key, config_value, config_type, description) VALUES
('max_file_size', '10485760', 'number', '最大文件上传大小（字节）'),
('supported_file_types', '["pdf","doc","docx","txt","md"]', 'json', '支持的文件类型'),
('daily_free_quota', '3', 'number', '每日免费生成课件配额'),
('vip_monthly_quota', '50', 'number', '月度VIP用户每日配额'),
('vip_yearly_quota', '100', 'number', '年度VIP用户每日配额'),
('ai_api_key', '', 'string', 'AI API密钥'),
('ai_api_url', 'https://dashscope.aliyuncs.com/api/v1', 'string', 'AI API地址'),
('tts_access_key_id', '', 'string', 'TTS访问密钥ID'),
('tts_access_key_secret', '', 'string', 'TTS访问密钥'),
('max_slides_per_course', '50', 'number', '每个课件最大幻灯片数'),
('max_questions_per_quiz', '20', 'number', '每个练习最大题目数'),
('course_generation_timeout', '300', 'number', '课件生成超时时间（秒）'),
('audio_cdn_base_url', '', 'string', '音频CDN基础URL'),
('share_link_expire_days', '30', 'number', '分享链接默认过期天数'),
('vip_monthly_price', '1980', 'number', '月度VIP价格（分）'),
('vip_yearly_price', '9900', 'number', '年度VIP价格（分）');

-- 2. 插入测试用户数据
INSERT INTO users (openid, nickname, avatar_url, phone, email, vip_level, vip_expired_at, credits, total_courses_created, total_study_time) VALUES
('wx_test_001', '张三', 'https://example.com/avatar1.jpg', '13812345678', 'zhangsan@example.com', 0, NULL, 10, 0, 0),
('wx_test_002', '李四', 'https://example.com/avatar2.jpg', '13887654321', 'lisi@example.com', 1, '2024-08-17 23:59:59', 50, 2, 1800),
('wx_test_003', '王五', 'https://example.com/avatar3.jpg', '13666666666', 'wangwu@example.com', 2, '2025-07-17 23:59:59', 100, 5, 3600),
('wx_test_004', '赵六', 'https://example.com/avatar4.jpg', '13555555555', 'zhaoliu@example.com', 0, NULL, 8, 1, 600);

-- 3. 插入测试课件数据
INSERT INTO courses (user_id, title, description, category, tags, source_type, source_content, source_url, status, slides_count, duration, view_count, like_count, is_public, voice_type) VALUES
(2, 'Go语言基础教程', '从零开始学习Go语言的基础知识，包括语法、数据类型、函数等核心概念。', 'programming', '["Go", "编程", "后端"]', 'text', 'Go语言是Google开发的开源编程语言...', NULL, 'completed', 8, 480, 125, 18, true, 'xiaoyun'),
(2, 'MySQL数据库设计', '学习如何设计高效的MySQL数据库，包括表结构设计、索引优化等。', 'database', '["MySQL", "数据库", "设计"]', 'url', 'MySQL是最流行的关系型数据库...', 'https://dev.mysql.com/doc/', 'completed', 12, 720, 89, 12, true, 'xiaogang'),
(3, 'React前端开发实战', 'React框架的实战开发教程，包括组件设计、状态管理、路由等。', 'frontend', '["React", "前端", "JavaScript"]', 'document', 'React是Facebook开发的前端框架...', NULL, 'completed', 15, 900, 203, 35, true, 'xiaoyun'),
(3, '机器学习入门', '机器学习的基础概念和常用算法介绍，适合初学者。', 'ai', '["机器学习", "AI", "算法"]', 'text', '机器学习是人工智能的重要分支...', NULL, 'generating', 0, 0, 0, 0, false, 'xiaoyun'),
(4, 'Docker容器化部署', '学习如何使用Docker进行应用容器化和部署。', 'devops', '["Docker", "容器", "部署"]', 'url', 'Docker是一个开源的容器化平台...', 'https://docs.docker.com/', 'completed', 10, 600, 56, 8, false, 'xiaogang');

-- 4. 插入幻灯片数据
INSERT INTO slides (course_id, slide_number, title, content, speaker_notes, audio_url, duration, layout_type) VALUES
-- Go语言基础教程 (course_id: 1)
(1, 1, 'Go语言简介', 'Go语言是Google开发的开源编程语言，具有高并发、高性能的特点。', '欢迎大家学习Go语言，这是一门非常优秀的编程语言。', '/audio/course1/slide1.mp3', 60, 'title'),
(1, 2, 'Go语言特性', '1. 静态编译 2. 垃圾回收 3. 并发支持 4. 简洁语法', 'Go语言有四个主要特性，让我们一一了解。', '/audio/course1/slide2.mp3', 70, 'content'),
(1, 3, '安装Go环境', '从官网下载Go安装包，配置GOPATH和GOROOT环境变量。', '安装Go环境是学习的第一步，请大家仔细操作。', '/audio/course1/slide3.mp3', 55, 'content'),
(1, 4, '第一个Go程序', 'package main\nimport "fmt"\nfunc main() {\n    fmt.Println("Hello, World!")\n}', '这是最经典的Hello World程序，展示了Go的基本结构。', '/audio/course1/slide4.mp3', 65, 'content'),

-- MySQL数据库设计 (course_id: 2)
(2, 1, 'MySQL数据库设计', '学习如何设计高效、规范的MySQL数据库结构。', '数据库设计是系统开发的基础，非常重要。', '/audio/course2/slide1.mp3', 50, 'title'),
(2, 2, '数据库设计原则', '1. 规范化原则 2. 性能优化 3. 可维护性 4. 扩展性', '好的数据库设计需要遵循这些基本原则。', '/audio/course2/slide2.mp3', 75, 'content'),
(2, 3, '表结构设计', '合理选择数据类型，设计主键和外键关系。', '表结构设计直接影响数据库的性能和维护性。', '/audio/course2/slide3.mp3', 68, 'content'),

-- React前端开发实战 (course_id: 3)
(3, 1, 'React开发实战', '通过实战项目学习React框架的核心概念和最佳实践。', 'React是目前最流行的前端框架之一。', '/audio/course3/slide1.mp3', 45, 'title'),
(3, 2, 'React组件', '组件是React的核心概念，包括函数组件和类组件。', '组件化开发让前端开发更加模块化和可维护。', '/audio/course3/slide2.mp3', 80, 'content');

-- 5. 插入练习数据
INSERT INTO quizzes (course_id, title, description, total_questions, total_points, time_limit, pass_score) VALUES
(1, 'Go语言基础测试', '测试对Go语言基础知识的掌握程度', 5, 10, 30, 6),
(2, 'MySQL设计能力测试', '测试数据库设计相关知识', 4, 8, 25, 5),
(3, 'React基础测试', '测试React基础概念理解', 6, 12, 35, 7);

-- 6. 插入题目数据
INSERT INTO questions (quiz_id, course_id, slide_id, type, question, options, correct_answer, explanation, difficulty, points, order_num) VALUES
-- Go语言基础测试题目
(1, 1, 1, 'single_choice', 'Go语言是由哪家公司开发的？', '["A. Microsoft", "B. Google", "C. Facebook", "D. Apple"]', 'B', 'Go语言是由Google公司开发的开源编程语言。', 'easy', 2, 1),
(1, 1, 2, 'multiple_choice', 'Go语言的特性包括哪些？（多选）', '["A. 静态编译", "B. 垃圾回收", "C. 并发支持", "D. 动态类型"]', 'A,B,C', 'Go语言支持静态编译、垃圾回收和并发，但是静态类型语言。', 'normal', 3, 2),
(1, 1, 3, 'true_false', 'Go语言需要手动管理内存。', '["true", "false"]', 'false', 'Go语言有自动垃圾回收机制，不需要手动管理内存。', 'easy', 2, 3),
(1, 1, 4, 'fill_blank', '在Go语言中，程序的入口函数是____。', '[]', 'main', 'Go程序的入口函数是main函数。', 'easy', 2, 4),
(1, 1, NULL, 'single_choice', 'Go语言的文件扩展名是什么？', '["A. .go", "B. .golang", "C. .g", "D. .gol"]', 'A', 'Go语言源代码文件的扩展名是.go。', 'easy', 1, 5),

-- MySQL设计能力测试题目
(2, 2, 2, 'single_choice', '数据库设计的第一范式要求是什么？', '["A. 消除重复", "B. 原子性", "C. 一致性", "D. 隔离性"]', 'B', '第一范式要求每个字段都是原子性的，不可再分。', 'normal', 2, 1),
(2, 2, 3, 'multiple_choice', '设计数据库主键时应该考虑哪些因素？', '["A. 唯一性", "B. 稳定性", "C. 简洁性", "D. 可读性"]', 'A,B,C', '主键应该具备唯一性、稳定性和简洁性。', 'normal', 3, 2),
(2, 2, NULL, 'true_false', '外键约束可以保证数据的参照完整性。', '["true", "false"]', 'true', '外键约束确实可以保证数据的参照完整性。', 'easy', 2, 3),
(2, 2, NULL, 'fill_blank', 'MySQL中自动递增的字段属性是____。', '[]', 'AUTO_INCREMENT', 'AUTO_INCREMENT是MySQL中实现自动递增的字段属性。', 'easy', 1, 4),

-- React基础测试题目
(3, 3, 1, 'single_choice', 'React是由哪家公司开发的？', '["A. Google", "B. Microsoft", "C. Facebook", "D. Twitter"]', 'C', 'React是由Facebook（现Meta）公司开发的。', 'easy', 2, 1),
(3, 3, 2, 'multiple_choice', 'React组件可以分为哪几种类型？', '["A. 函数组件", "B. 类组件", "C. 高阶组件", "D. 原生组件"]', 'A,B', 'React主要有函数组件和类组件两种基本类型。', 'normal', 3, 2),
(3, 3, 2, 'true_false', 'React组件必须返回单个根元素。', '["true", "false"]', 'false', 'React 16+支持Fragment，可以返回多个元素。', 'normal', 2, 3),
(3, 3, NULL, 'fill_blank', 'React中用于管理组件状态的Hook是____。', '[]', 'useState', 'useState是React中最常用的状态管理Hook。', 'normal', 2, 4),
(3, 3, NULL, 'single_choice', 'JSX是什么？', '["A. JavaScript扩展", "B. Java语法", "C. JSON格式", "D. XML标准"]', 'A', 'JSX是JavaScript的语法扩展，用于描述UI结构。', 'easy', 2, 5),
(3, 3, NULL, 'single_choice', 'React的虚拟DOM有什么作用？', '["A. 减少内存使用", "B. 提高渲染性能", "C. 简化语法", "D. 增加安全性"]', 'B', '虚拟DOM通过diff算法优化DOM操作，提高渲染性能。', 'normal', 1, 6);

-- 7. 插入学习记录数据
INSERT INTO learning_records (user_id, course_id, current_slide, total_slides, progress, study_duration, complete_rate, is_completed, status, start_time, last_study_time) VALUES
(2, 1, 8, 8, 100.00, 480, 100.00, true, 'completed', '2024-07-15 09:30:00', '2024-07-15 10:15:00'),
(2, 2, 6, 12, 50.00, 320, 50.00, false, 'learning', '2024-07-16 14:20:00', '2024-07-16 15:45:00'),
(3, 1, 4, 8, 50.00, 240, 50.00, false, 'paused', '2024-07-14 16:10:00', '2024-07-14 17:00:00'),
(4, 1, 3, 8, 37.50, 180, 37.50, false, 'learning', '2024-07-17 08:45:00', '2024-07-17 09:30:00');

-- 8. 插入答题记录数据
INSERT INTO quiz_attempts (user_id, quiz_id, course_id, start_time, end_time, duration, total_score, user_score, percentage, is_passed, is_completed, correct_count, wrong_count, submitted_at) VALUES
(2, 1, 1, '2024-07-15 10:20:00', '2024-07-15 10:35:00', 900, 10, 8, 80.00, true, true, 4, 1, '2024-07-15 10:35:00'),
(3, 1, 1, '2024-07-16 11:15:00', '2024-07-16 11:28:00', 780, 10, 6, 60.00, true, true, 3, 2, '2024-07-16 11:28:00'),
(4, 1, 1, '2024-07-17 09:45:00', NULL, 0, 10, 0, 0, false, false, 0, 0, NULL);

-- 9. 插入用户答案数据
INSERT INTO user_answers (user_id, question_id, attempt_id, user_answer, is_correct, points, duration) VALUES
-- 用户2的答题记录
(2, 1, 1, 'B', true, 2, 45),
(2, 2, 1, 'A,B,C', true, 3, 120),
(2, 3, 1, 'false', true, 2, 30),
(2, 4, 1, 'main', true, 2, 35),
(2, 5, 1, 'B', false, 0, 25),

-- 用户3的答题记录
(3, 1, 2, 'B', true, 2, 55),
(3, 2, 2, 'A,B', false, 0, 140),
(3, 3, 2, 'false', true, 2, 25),
(3, 4, 2, 'main', true, 2, 40),
(3, 5, 2, 'A', true, 1, 30);

-- 10. 插入分享记录数据
INSERT INTO share_records (user_id, course_id, share_code, title, description, view_count, like_count, download_count, expires_at) VALUES
(2, 1, 'go_basics_2024_001', 'Go语言基础教程', '从零开始学习Go语言，适合初学者', 45, 8, 12, '2024-08-17 23:59:59'),
(3, 3, 'react_tutorial_003', 'React前端开发实战', 'React框架实战教程，包含项目案例', 78, 15, 23, '2024-08-17 23:59:59');

-- 11. 插入分享点赞数据
INSERT INTO share_likes (share_id, user_id) VALUES
(1, 3), (1, 4),
(2, 2), (2, 4);

-- 12. 插入使用记录数据
INSERT INTO usage_records (user_id, type, resource_id, consumed_credits, ip_address, user_agent) VALUES
(2, 'course_generation', 1, 1, '192.168.1.100', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'),
(2, 'course_generation', 2, 1, '192.168.1.100', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'),
(3, 'course_generation', 3, 1, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'),
(3, 'tts_generation', 3, 1, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'),
(4, 'course_generation', 5, 1, '192.168.1.102', 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)');

-- 13. 插入购买记录数据
INSERT INTO purchase_records (user_id, order_no, plan_id, plan_name, amount, payment_method, status, paid_at) VALUES
(2, 'ORDER_202407150001', 1, '月度VIP会员', 1980, 'wechat', 'completed', '2024-07-15 12:30:00'),
(3, 'ORDER_202407100001', 2, '年度VIP会员', 9900, 'wechat', 'completed', '2024-07-10 15:45:00');

-- 14. 插入音频元数据
INSERT INTO audio_metadata (course_id, slide_id, file_name, file_path, file_size, duration, format, url, status) VALUES
(1, 1, 'course1_slide1.mp3', '/storage/audio/course1/slide1.mp3', 485760, 60, 'mp3', '/api/audio/course1/slide1.mp3', 'completed'),
(1, 2, 'course1_slide2.mp3', '/storage/audio/course1/slide2.mp3', 563200, 70, 'mp3', '/api/audio/course1/slide2.mp3', 'completed'),
(1, 3, 'course1_slide3.mp3', '/storage/audio/course1/slide3.mp3', 442880, 55, 'mp3', '/api/audio/course1/slide3.mp3', 'completed'),
(1, 4, 'course1_slide4.mp3', '/storage/audio/course1/slide4.mp3', 524160, 65, 'mp3', '/api/audio/course1/slide4.mp3', 'completed'),
(2, 5, 'course2_slide1.mp3', '/storage/audio/course2/slide1.mp3', 403200, 50, 'mp3', '/api/audio/course2/slide1.mp3', 'completed'),
(2, 6, 'course2_slide2.mp3', '/storage/audio/course2/slide2.mp3', 604800, 75, 'mp3', '/api/audio/course2/slide2.mp3', 'completed'),
(2, 7, 'course2_slide3.mp3', '/storage/audio/course2/slide3.mp3', 548160, 68, 'mp3', '/api/audio/course2/slide3.mp3', 'completed'),
(3, 8, 'course3_slide1.mp3', '/storage/audio/course3/slide1.mp3', 363000, 45, 'mp3', '/api/audio/course3/slide1.mp3', 'completed'),
(3, 9, 'course3_slide2.mp3', '/storage/audio/course3/slide2.mp3', 645120, 80, 'mp3', '/api/audio/course3/slide2.mp3', 'completed');

-- 15. 插入操作日志数据
INSERT INTO operation_logs (user_id, operation_type, operation_desc, resource_type, resource_id, ip_address, user_agent, execution_time, status) VALUES
(2, 'user_login', '用户微信登录', 'user', 2, '192.168.1.100', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', 350, 'success'),
(2, 'course_create', '创建课件：Go语言基础教程', 'course', 1, '192.168.1.100', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', 2800, 'success'),
(3, 'user_login', '用户微信登录', 'user', 3, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 280, 'success'),
(3, 'course_create', '创建课件：React前端开发实战', 'course', 3, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 3200, 'success'),
(4, 'user_login', '用户微信登录', 'user', 4, '192.168.1.102', 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0)', 420, 'success'),
(2, 'quiz_attempt', '开始答题测试', 'quiz', 1, '192.168.1.100', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', 150, 'success'),
(3, 'course_share', '分享课件', 'course', 3, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 250, 'success');

-- =============================================
-- 创建完成提示
-- =============================================

SELECT 'AI课堂数据库初始化完成！' AS message;
SELECT '数据统计：' AS info;
SELECT COUNT(*) AS users_count FROM users;
SELECT COUNT(*) AS courses_count FROM courses;
SELECT COUNT(*) AS slides_count FROM slides;
SELECT COUNT(*) AS questions_count FROM questions;
SELECT COUNT(*) AS system_configs_count FROM system_configs; 