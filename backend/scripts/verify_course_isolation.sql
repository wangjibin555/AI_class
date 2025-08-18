-- 验证课件数据隔离效果的SQL查询

-- 1. 检查课件与用户的关联关系
SELECT 
    c.id as course_id,
    c.title,
    c.user_id,
    u.openid,
    u.nickname,
    c.is_public,
    c.status,
    c.created_at
FROM courses c
LEFT JOIN users u ON c.user_id = u.id
ORDER BY c.created_at DESC
LIMIT 20;

-- 2. 统计每个用户的课件数量
SELECT 
    u.id as user_id,
    u.openid,
    u.nickname,
    COUNT(c.id) as course_count,
    COUNT(CASE WHEN c.is_public = 1 THEN 1 END) as public_courses,
    COUNT(CASE WHEN c.is_public = 0 THEN 1 END) as private_courses
FROM users u
LEFT JOIN courses c ON u.id = c.user_id
GROUP BY u.id, u.openid, u.nickname
HAVING course_count > 0
ORDER BY course_count DESC;

-- 3. 检查是否有孤儿课件（user_id不存在的课件）
SELECT 
    c.id,
    c.title,
    c.user_id,
    c.created_at
FROM courses c
LEFT JOIN users u ON c.user_id = u.id
WHERE u.id IS NULL;

-- 4. 检查公开课件（其他用户可访问）
SELECT 
    c.id,
    c.title,
    u.nickname as creator,
    c.view_count,
    c.like_count,
    c.created_at
FROM courses c
JOIN users u ON c.user_id = u.id
WHERE c.is_public = 1
ORDER BY c.view_count DESC;

-- 5. 检查学习记录的用户隔离
SELECT 
    lr.id,
    lr.user_id,
    u.nickname as learner,
    lr.course_id,
    c.title as course_title,
    cu.nickname as course_creator,
    lr.progress,
    lr.is_completed
FROM learning_records lr
JOIN users u ON lr.user_id = u.id
JOIN courses c ON lr.course_id = c.id
JOIN users cu ON c.user_id = cu.id
ORDER BY lr.created_at DESC
LIMIT 10;

-- 6. 验证用户只能看到自己的私有课件
-- 模拟用户ID=1的查询
SELECT 
    c.id,
    c.title,
    c.user_id,
    c.is_public,
    CASE 
        WHEN c.user_id = 1 THEN '可访问（自己的）'
        WHEN c.is_public = 1 THEN '可访问（公开的）'
        ELSE '不可访问（他人私有）'
    END as access_status
FROM courses c
ORDER BY c.id;
