-- 修复数据库中的音频URL格式
-- 将所有 .wav 扩展名改为 .mp3

-- 更新 slides 表中的 audio_url 字段
UPDATE slides 
SET audio_url = REPLACE(audio_url, '.wav', '.mp3')
WHERE audio_url LIKE '%.wav';

-- 更新 audio_metadata 表中的相关字段（如果存在）
UPDATE audio_metadata 
SET 
    file_name = REPLACE(file_name, '.wav', '.mp3'),
    file_path = REPLACE(file_path, '.wav', '.mp3'),
    url = REPLACE(url, '.wav', '.mp3'),
    cdn_url = REPLACE(cdn_url, '.wav', '.mp3'),
    format = 'mp3'
WHERE format = 'wav' OR file_name LIKE '%.wav' OR file_path LIKE '%.wav' OR url LIKE '%.wav' OR cdn_url LIKE '%.wav';

-- 显示更新结果
SELECT 
    COUNT(*) as total_slides_with_audio,
    COUNT(CASE WHEN audio_url LIKE '%.mp3' THEN 1 END) as mp3_slides,
    COUNT(CASE WHEN audio_url LIKE '%.wav' THEN 1 END) as wav_slides
FROM slides 
WHERE audio_url IS NOT NULL AND audio_url != '';

-- 如果有 audio_metadata 表，也显示其统计
SELECT 
    COUNT(*) as total_audio_metadata,
    COUNT(CASE WHEN format = 'mp3' THEN 1 END) as mp3_format,
    COUNT(CASE WHEN format = 'wav' THEN 1 END) as wav_format
FROM audio_metadata;