-- 创建练习表
CREATE TABLE `quizzes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '练习ID',
  `course_id` bigint unsigned NOT NULL COMMENT '课件ID',
  `title` varchar(200) NOT NULL COMMENT '练习标题',
  `description` text COMMENT '练习描述',
  `total_questions` int unsigned NOT NULL DEFAULT '0' COMMENT '题目总数',
  `total_points` int unsigned NOT NULL DEFAULT '0' COMMENT '总分',
  `time_limit` int unsigned DEFAULT NULL COMMENT '时间限制(分钟)',
  `pass_score` int unsigned DEFAULT '60' COMMENT '及格分',
  `attempt_limit` int unsigned DEFAULT '3' COMMENT '答题次数限制',
  `is_random_order` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否随机题目顺序',
  `show_correct_answer` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '是否显示正确答案',
  `status` enum('draft','active','inactive') NOT NULL DEFAULT 'active' COMMENT '状态',
  `difficulty_level` enum('easy','normal','hard') DEFAULT 'normal' COMMENT '难度等级',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_course_id` (`course_id`),
  KEY `idx_status` (`status`),
  CONSTRAINT `fk_quizzes_course_id` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='练习表';

-- 创建题目表
CREATE TABLE `questions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '题目ID',
  `quiz_id` bigint unsigned NOT NULL COMMENT '练习ID',
  `type` enum('single_choice','multiple_choice','true_false','fill_blank','short_answer') NOT NULL COMMENT '题目类型',
  `question` longtext NOT NULL COMMENT '题目内容',
  `options` json DEFAULT NULL COMMENT '选项(JSON数组)',
  `correct_answer` text NOT NULL COMMENT '正确答案',
  `explanation` text COMMENT '答案解析',
  `points` int unsigned NOT NULL DEFAULT '10' COMMENT '分值',
  `difficulty` enum('easy','normal','hard') DEFAULT 'normal' COMMENT '难度',
  `order_num` int unsigned NOT NULL COMMENT '题目顺序',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_quiz_id` (`quiz_id`),
  KEY `idx_type` (`type`),
  KEY `idx_difficulty` (`difficulty`),
  CONSTRAINT `fk_questions_quiz_id` FOREIGN KEY (`quiz_id`) REFERENCES `quizzes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='题目表';

-- 创建答题记录表
CREATE TABLE `quiz_attempts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '答题记录ID',
  `quiz_id` bigint unsigned NOT NULL COMMENT '练习ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `attempt_number` int unsigned NOT NULL COMMENT '第几次尝试',
  `total_score` int unsigned NOT NULL DEFAULT '0' COMMENT '总分',
  `user_score` int unsigned NOT NULL DEFAULT '0' COMMENT '用户得分',
  `percentage` decimal(5,2) NOT NULL DEFAULT '0.00' COMMENT '得分百分比',
  `correct_count` int unsigned NOT NULL DEFAULT '0' COMMENT '正确题目数',
  `wrong_count` int unsigned NOT NULL DEFAULT '0' COMMENT '错误题目数',
  `duration` int unsigned DEFAULT NULL COMMENT '答题用时(秒)',
  `is_passed` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否通过',
  `status` enum('in_progress','completed','timeout','cancelled') NOT NULL DEFAULT 'in_progress' COMMENT '状态',
  `start_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '开始时间',
  `end_time` timestamp NULL DEFAULT NULL COMMENT '结束时间',
  `submitted_at` timestamp NULL DEFAULT NULL COMMENT '提交时间',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_quiz_user_attempt` (`quiz_id`, `user_id`, `attempt_number`),
  KEY `idx_quiz_id` (`quiz_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  CONSTRAINT `fk_quiz_attempts_quiz_id` FOREIGN KEY (`quiz_id`) REFERENCES `quizzes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_quiz_attempts_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='答题记录表';

-- 创建用户答案表
CREATE TABLE `user_answers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '答案ID',
  `attempt_id` bigint unsigned NOT NULL COMMENT '答题记录ID',
  `question_id` bigint unsigned NOT NULL COMMENT '题目ID',
  `user_answer` text COMMENT '用户答案',
  `is_correct` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否正确',
  `points_earned` int unsigned NOT NULL DEFAULT '0' COMMENT '获得分数',
  `duration` int unsigned DEFAULT NULL COMMENT '答题用时(秒)',
  `answered_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '答题时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_attempt_question` (`attempt_id`, `question_id`),
  KEY `idx_attempt_id` (`attempt_id`),
  KEY `idx_question_id` (`question_id`),
  CONSTRAINT `fk_user_answers_attempt_id` FOREIGN KEY (`attempt_id`) REFERENCES `quiz_attempts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_answers_question_id` FOREIGN KEY (`question_id`) REFERENCES `questions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户答案表';