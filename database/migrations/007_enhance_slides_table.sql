-- 增强slides表结构，添加关键词和要点字段
-- 迁移文件：007_enhance_slides_table.sql
-- 创建时间：2025-07-23

-- 添加要点列表字段（如果不存在）
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.columns 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND column_name = 'bullet_points';

SET @sql = IF(@col_exists = 0, 
  'ALTER TABLE slides ADD COLUMN bullet_points JSON COMMENT "幻灯片要点列表，JSON数组格式"', 
  'SELECT "bullet_points column already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加幻灯片类型字段（如果不存在）
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.columns 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND column_name = 'slide_type';

SET @sql = IF(@col_exists = 0, 
  'ALTER TABLE slides ADD COLUMN slide_type VARCHAR(20) DEFAULT "content" COMMENT "幻灯片类型：title, content, summary, transition"', 
  'SELECT "slide_type column already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加预计时长字段（如果不存在）
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.columns 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND column_name = 'estimated_time';

SET @sql = IF(@col_exists = 0, 
  'ALTER TABLE slides ADD COLUMN estimated_time INT DEFAULT 60 COMMENT "预计演讲时间（秒）"', 
  'SELECT "estimated_time column already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加重要程度字段（如果不存在）
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.columns 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND column_name = 'importance';

SET @sql = IF(@col_exists = 0, 
  'ALTER TABLE slides ADD COLUMN importance INT DEFAULT 3 COMMENT "重要程度：1-5"', 
  'SELECT "importance column already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加图片建议字段（如果不存在）
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.columns 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND column_name = 'image_suggestion';

SET @sql = IF(@col_exists = 0, 
  'ALTER TABLE slides ADD COLUMN image_suggestion TEXT COMMENT "图片建议描述"', 
  'SELECT "image_suggestion column already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 创建索引（如果不存在）
SET @index_exists = 0;
SELECT COUNT(*) INTO @index_exists 
FROM information_schema.statistics 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND index_name = 'idx_slides_type';

SET @sql = IF(@index_exists = 0, 
  'CREATE INDEX idx_slides_type ON slides (slide_type)', 
  'SELECT "idx_slides_type index already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists = 0;
SELECT COUNT(*) INTO @index_exists 
FROM information_schema.statistics 
WHERE table_schema = 'ai_classroom' 
  AND table_name = 'slides' 
  AND index_name = 'idx_slides_importance';

SET @sql = IF(@index_exists = 0, 
  'CREATE INDEX idx_slides_importance ON slides (importance)', 
  'SELECT "idx_slides_importance index already exists" as msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 更新现有记录的默认值（只更新没有值的记录）
-- 先创建临时表存储每个课程的最大幻灯片编号
CREATE TEMPORARY TABLE temp_max_slides AS 
SELECT course_id, MAX(slide_number) as max_slide_number 
FROM slides 
GROUP BY course_id;

-- 更新bullet_points
UPDATE slides 
SET bullet_points = JSON_ARRAY()
WHERE bullet_points IS NULL;

-- 更新estimated_time
UPDATE slides 
SET estimated_time = CASE 
    WHEN LENGTH(COALESCE(content, '')) > 200 THEN 90
    WHEN LENGTH(COALESCE(content, '')) > 100 THEN 60
    ELSE 45
END
WHERE estimated_time IS NULL;

-- 更新slide_type
UPDATE slides s
JOIN temp_max_slides t ON s.course_id = t.course_id
SET s.slide_type = CASE 
    WHEN s.slide_number = 1 THEN 'title'
    WHEN s.slide_number = t.max_slide_number THEN 'summary'
        ELSE 'content'
END
WHERE s.slide_type IS NULL;

-- 更新importance
UPDATE slides s
JOIN temp_max_slides t ON s.course_id = t.course_id
SET s.importance = CASE 
    WHEN s.slide_number = 1 THEN 5
    WHEN s.slide_number = t.max_slide_number THEN 4
        ELSE 3
    END
WHERE s.importance IS NULL;

-- 删除临时表
DROP TEMPORARY TABLE temp_max_slides; 