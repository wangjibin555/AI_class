-- 验证用户隔离修复效果的SQL查询
-- 使用此脚本检查修复后的用户数据状态

-- 1. 检查所有用户的OpenID分布
SELECT 
    openid,
    nickname,
    id,
    credits,
    created_at,
    CASE 
        WHEN openid LIKE 'mock_openid%' THEN '测试用户'
        ELSE '真实用户'
    END as user_type
FROM users 
ORDER BY created_at DESC;

-- 2. 统计用户类型分布
SELECT 
    CASE 
        WHEN openid LIKE 'mock_openid%' THEN '测试用户'
        ELSE '真实用户'
    END as user_type,
    COUNT(*) as count
FROM users 
GROUP BY 
    CASE 
        WHEN openid LIKE 'mock_openid%' THEN '测试用户'
        ELSE '真实用户'
    END;

-- 3. 检查重复的OpenID（应该没有）
SELECT 
    openid, 
    COUNT(*) as duplicate_count 
FROM users 
GROUP BY openid 
HAVING COUNT(*) > 1;

-- 4. 检查是否还有旧的共享测试用户数据
SELECT * FROM users 
WHERE openid = 'mock_openid_12345' 
   OR nickname = '测试用户';

-- 5. 查看最近登录的用户
SELECT 
    id,
    openid,
    nickname,
    credits,
    created_at,
    updated_at
FROM users 
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 DAY)
ORDER BY created_at DESC;

-- 6. 检查课程数据的用户关联
SELECT 
    c.title,
    c.user_id,
    u.openid,
    u.nickname,
    c.created_at
FROM courses c
LEFT JOIN users u ON c.user_id = u.id
ORDER BY c.created_at DESC
LIMIT 10;
