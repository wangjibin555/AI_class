-- 创建学习记录表
CREATE TABLE `learning_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '学习记录ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `course_id` bigint unsigned NOT NULL COMMENT '课件ID',
  `current_slide` int unsigned NOT NULL DEFAULT '1' COMMENT '当前幻灯片',
  `total_slides` int unsigned NOT NULL DEFAULT '0' COMMENT '总幻灯片数',
  `progress` decimal(5,2) NOT NULL DEFAULT '0.00' COMMENT '学习进度百分比',
  `study_duration` int unsigned NOT NULL DEFAULT '0' COMMENT '学习时长(秒)',
  `last_position` int unsigned DEFAULT '0' COMMENT '最后播放位置(秒)',
  `is_completed` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否完成',
  `is_bookmarked` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否收藏',
  `completion_rate` decimal(5,2) DEFAULT '0.00' COMMENT '完成率',
  `quiz_best_score` decimal(5,2) DEFAULT NULL COMMENT '练习最高分',
  `quiz_attempts` int unsigned NOT NULL DEFAULT '0' COMMENT '练习尝试次数',
  `status` enum('not_started','learning','paused','completed') NOT NULL DEFAULT 'not_started' COMMENT '学习状态',
  `start_time` timestamp NULL DEFAULT NULL COMMENT '开始学习时间',
  `last_study_time` timestamp NULL DEFAULT NULL COMMENT '最后学习时间',
  `completed_at` timestamp NULL DEFAULT NULL COMMENT '完成时间',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_course` (`user_id`, `course_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_course_id` (`course_id`),
  KEY `idx_status` (`status`),
  CONSTRAINT `fk_learning_records_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_learning_records_course_id` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='学习记录表';

-- 创建分享记录表
CREATE TABLE `shares` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '分享ID',
  `user_id` bigint unsigned NOT NULL COMMENT '分享用户ID',
  `course_id` bigint unsigned NOT NULL COMMENT '课件ID',
  `share_code` varchar(32) NOT NULL COMMENT '分享码',
  `title` varchar(200) DEFAULT NULL COMMENT '分享标题',
  `description` text COMMENT '分享描述',
  `cover_image` varchar(500) DEFAULT NULL COMMENT '封面图片',
  `view_count` int unsigned NOT NULL DEFAULT '0' COMMENT '浏览次数',
  `like_count` int unsigned NOT NULL DEFAULT '0' COMMENT '点赞数',
  `download_count` int unsigned NOT NULL DEFAULT '0' COMMENT '下载次数',
  `allow_download` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '是否允许下载',
  `is_active` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '是否激活',
  `expires_at` timestamp NULL DEFAULT NULL COMMENT '过期时间',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_share_code` (`share_code`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_course_id` (`course_id`),
  KEY `idx_is_active` (`is_active`),
  CONSTRAINT `fk_shares_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_shares_course_id` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分享记录表';

-- 创建VIP订单表
CREATE TABLE `vip_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '订单ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `order_no` varchar(64) NOT NULL COMMENT '订单号',
  `plan_id` int unsigned NOT NULL COMMENT '套餐ID',
  `plan_name` varchar(100) NOT NULL COMMENT '套餐名称',
  `amount` int unsigned NOT NULL COMMENT '金额(分)',
  `payment_method` enum('wechat','alipay') NOT NULL COMMENT '支付方式',
  `status` enum('pending','paid','failed','refunded','cancelled') NOT NULL DEFAULT 'pending' COMMENT '订单状态',
  `transaction_id` varchar(64) DEFAULT NULL COMMENT '第三方交易号',
  `payment_info` json DEFAULT NULL COMMENT '支付信息',
  `paid_at` timestamp NULL DEFAULT NULL COMMENT '支付时间',
  `refunded_at` timestamp NULL DEFAULT NULL COMMENT '退款时间',
  `refund_reason` varchar(255) DEFAULT NULL COMMENT '退款原因',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_order_no` (`order_no`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_transaction_id` (`transaction_id`),
  CONSTRAINT `fk_vip_orders_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='VIP订单表';

-- 创建用户使用记录表
CREATE TABLE `usage_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `type` enum('course_generation','tts_generation','quiz_generation','share','download') NOT NULL COMMENT '使用类型',
  `resource_id` bigint unsigned DEFAULT NULL COMMENT '关联资源ID',
  `consumed_credits` int unsigned NOT NULL DEFAULT '0' COMMENT '消耗积分',
  `metadata` json DEFAULT NULL COMMENT '元数据',
  `ip_address` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `user_agent` varchar(500) DEFAULT NULL COMMENT '用户代理',
  `consumed_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '消耗时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_type` (`type`),
  KEY `idx_consumed_at` (`consumed_at`),
  CONSTRAINT `fk_usage_records_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户使用记录表';

-- 创建系统配置表
CREATE TABLE `system_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '配置ID',
  `config_key` varchar(100) NOT NULL COMMENT '配置键',
  `config_value` longtext COMMENT '配置值',
  `config_type` enum('string','number','boolean','json') NOT NULL DEFAULT 'string' COMMENT '配置类型',
  `description` varchar(255) DEFAULT NULL COMMENT '配置描述',
  `is_public` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否公开',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';