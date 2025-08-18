#!/usr/bin/env node

/**
 * 音频生成和DashScope PPT生成功能测试脚本
 * 测试完整的生成流程：创建课件 -> 生成音频/DashScope PPT -> 检查状态
 */

const https = require('https');
const { URL } = require('url');
const fs = require('fs');

// 配置
const CONFIG = {
    BASE_URL: 'https://wangjibin-sc.wepie.com:9000',
    TIMEOUT_MS: 300000, // 5分钟超时
    POLL_INTERVAL_MS: 10000, // 10秒轮询间隔
    MAX_POLL_ATTEMPTS: 30, // 最大轮询次数（5分钟）
    MOCK_USER_CREDENTIALS: {
        code: 'test_generation_' + Date.now(),
        userInfo: {
            nickName: '生成功能测试用户',
            avatarUrl: 'https://example.com/avatar.png'
        }
    }
};

// 测试数据
const TEST_DATA = {
    // 测试课件数据
    testCourse: {
        title: '音频PPT生成测试课件',
        description: '这是一个用于测试音频和PPT生成功能的测试课件。包含多个幻灯片内容，用于验证TTS和AI生成功能。',
        content: `
# 测试课件：AI技术概述

## 第一章：人工智能简介
人工智能（AI）是计算机科学的一个分支，致力于创建能够执行通常需要人类智能的任务的智能机器。

## 第二章：机器学习基础
机器学习是AI的一个子集，它使计算机能够在没有明确编程的情况下学习和改进。

## 第三章：深度学习
深度学习使用神经网络来模拟人脑的工作方式，用于图像识别、自然语言处理等领域。

## 第四章：应用场景
AI技术广泛应用于医疗诊断、自动驾驶、语音识别、推荐系统等各个领域。

## 结论
人工智能正在改变我们的世界，为未来带来无限可能。
        `,
        category: 'technology',
        voice_type: 'zhixiaobai',
        is_public: false
    },
    
    // DashScope PPT测试数据
    dashscopePPTTest: {
        content: 'AI人工智能发展历程及未来趋势分析',
        slides_count: 8,
        style: 'professional'
    },
    
    // URL分析测试
    urlAnalysisTest: {
        url: 'https://zh.wikipedia.org/wiki/%E4%BA%BA%E5%B7%A5%E6%99%BA%E8%83%BD',
        title: '人工智能 - 维基百科'
    }
};

// 测试统计
const testStats = {
    total: 0,
    successful: 0,
    failed: 0,
    timeouts: 0,
    results: [],
    createdResources: [] // 记录创建的资源以便清理
};

// HTTP请求工具函数
function makeRequest(options, data = null) {
    return new Promise((resolve) => {
        const startTime = Date.now();
        
        const req = https.request(options, (res) => {
            let responseData = '';
            
            res.on('data', chunk => responseData += chunk);
            res.on('end', () => {
                const endTime = Date.now();
                let parsedData = null;
                
                try {
                    parsedData = JSON.parse(responseData);
                } catch (e) {
                    // 忽略JSON解析错误
                }
                
                resolve({
                    statusCode: res.statusCode,
                    success: res.statusCode >= 200 && res.statusCode < 400,
                    data: parsedData,
                    rawData: responseData,
                    responseTime: endTime - startTime,
                    error: null
                });
            });
        });

        req.on('error', (error) => {
            resolve({
                statusCode: 0,
                success: false,
                data: null,
                rawData: null,
                responseTime: Date.now() - startTime,
                error: error.message
            });
        });

        req.on('timeout', () => {
            req.destroy();
            resolve({
                statusCode: 0,
                success: false,
                data: null,
                rawData: null,
                responseTime: Date.now() - startTime,
                error: 'Request timeout'
            });
        });

        if (data) {
            req.write(JSON.stringify(data));
        }
        
        req.end();
    });
}

// 创建HTTP请求选项
function createRequestOptions(path, method = 'GET', token = null) {
    const url = new URL(CONFIG.BASE_URL + path);
    
    const options = {
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'User-Agent': 'AudioPPTGenerationTest/1.0'
        },
        rejectUnauthorized: false,
        timeout: CONFIG.TIMEOUT_MS
    };

    if (token) {
        options.headers['Authorization'] = `Bearer ${token}`;
    }
    
    return options;
}

// 登录获取token
async function login() {
    console.log('🔐 登录获取认证token...');
    
    const options = createRequestOptions('/api/v1/auth/login', 'POST');
    const result = await makeRequest(options, CONFIG.MOCK_USER_CREDENTIALS);
    
    if (result.success && result.data) {
        const token = result.data.data?.token || result.data.token;
        if (token) {
            console.log('✅ 登录成功');
            return token;
        }
    }
    
    console.log('⚠️ 使用Mock token继续测试');
    return 'mock_token_' + Date.now();
}

// 创建测试课件
async function createTestCourse(token) {
    console.log('📚 创建测试课件...');
    
    const options = createRequestOptions('/api/v1/courses', 'POST', token);
    const result = await makeRequest(options, TEST_DATA.testCourse);
    
    testStats.total++;
    
    if (result.success && result.data) {
        testStats.successful++;
        const courseId = result.data.data?.course?.id || result.data.course?.id || result.data.id;
        
        if (courseId) {
            console.log(`✅ 课件创建成功，ID: ${courseId}`);
            testStats.createdResources.push({ type: 'course', id: courseId });
            return courseId;
        }
    }
    
    testStats.failed++;
    console.log(`❌ 课件创建失败: ${result.error || '未知错误'}`);
    console.log(`   响应: ${JSON.stringify(result.data)}`);
    return null;
}

// 测试音频生成
async function testAudioGeneration(courseId, token) {
    console.log(`🎵 测试课件 ${courseId} 的音频生成...`);
    
    // 1. 启动音频生成
    console.log('  📤 发起音频生成请求...');
    const generateOptions = createRequestOptions(`/api/v1/audio/generate/${courseId}`, 'POST', token);
    const generateResult = await makeRequest(generateOptions, {
        voice_type: 'zhixiaobai',
        speed: 1.0,
        volume: 50
    });
    
    testStats.total++;
    
    if (!generateResult.success) {
        testStats.failed++;
        console.log(`  ❌ 音频生成请求失败: ${generateResult.error || generateResult.statusCode}`);
        return false;
    }
    
    console.log(`  ✅ 音频生成请求成功 (${generateResult.responseTime}ms)`);
    
    // 2. 轮询状态
    console.log('  🔄 开始轮询音频生成状态...');
    
    for (let attempt = 1; attempt <= CONFIG.MAX_POLL_ATTEMPTS; attempt++) {
        await new Promise(resolve => setTimeout(resolve, CONFIG.POLL_INTERVAL_MS));
        
        const statusOptions = createRequestOptions(`/api/v1/audio/status/${courseId}`, 'GET', token);
        const statusResult = await makeRequest(statusOptions);
        
        testStats.total++;
        
        if (statusResult.success && statusResult.data) {
            const status = statusResult.data.status || statusResult.data.data?.status;
            console.log(`  📊 第${attempt}次检查 - 状态: ${status} (${statusResult.responseTime}ms)`);
            
            if (status === 'completed') {
                testStats.successful += 2; // 生成请求 + 状态检查
                console.log('  🎉 音频生成完成！');
                return true;
            } else if (status === 'failed' || status === 'error') {
                testStats.failed += 2;
                console.log('  ❌ 音频生成失败');
                return false;
            }
            
            // 继续轮询
        } else {
            console.log(`  ⚠️ 状态查询失败: ${statusResult.error || statusResult.statusCode}`);
        }
    }
    
    // 超时
    testStats.timeouts++;
    testStats.failed++;
    console.log('  ⏰ 音频生成超时');
    return false;
}

// 测试音频预览
async function testAudioPreview(token) {
    console.log('🎧 测试音频预览功能...');
    
    const options = createRequestOptions('/api/v1/audio/preview', 'POST', token);
    const result = await makeRequest(options, {
        text: '这是一个音频预览测试，用于验证TTS服务是否正常工作。',
        voice_type: 'zhixiaobai',
        speed: 1.0,
        volume: 50
    });
    
    testStats.total++;
    
    if (result.success) {
        testStats.successful++;
        console.log(`✅ 音频预览成功 (${result.responseTime}ms)`);
        return true;
    } else {
        testStats.failed++;
        console.log(`❌ 音频预览失败: ${result.error || result.statusCode}`);
        return false;
    }
}

// 测试DashScope PPT生成
async function testDashScopePPTGeneration(token) {
    console.log('📊 测试DashScope PPT生成...');
    
    // 1. 检查DashScope引擎状态
    console.log('  🔍 检查DashScope引擎状态...');
    const engineOptions = createRequestOptions('/api/v1/ai-engines', 'GET');
    const engineResult = await makeRequest(engineOptions);
    
    testStats.total++;
    
    if (!engineResult.success) {
        testStats.failed++;
        console.log(`  ❌ DashScope引擎状态检查失败: ${engineResult.error || engineResult.statusCode}`);
        return false;
    }
    
    console.log(`  ✅ DashScope引擎状态正常 (${engineResult.responseTime}ms)`);
    testStats.successful++;
    
    // 2. 测试增强PPT生成
    console.log('  📋 测试增强PPT生成...');
    const generateOptions = createRequestOptions('/api/v1/content/enhanced-ppt', 'POST', token);
    const generateResult = await makeRequest(generateOptions, {
        title: '人工智能发展趋势',
        content: TEST_DATA.dashscopePPTTest.content,
        slides_count: TEST_DATA.dashscopePPTTest.slides_count,
        style: TEST_DATA.dashscopePPTTest.style
    });
    
    testStats.total++;
    
    if (generateResult.success) {
        testStats.successful++;
        console.log(`  ✅ PPT生成请求成功 (${generateResult.responseTime}ms)`);
        
        // 检查响应数据
        if (generateResult.data) {
            console.log(`  📄 生成结果: ${JSON.stringify(generateResult.data).substring(0, 200)}...`);
        }
        
        return true;
    } else {
        testStats.failed++;
        console.log(`  ❌ PPT生成请求失败: ${generateResult.error || generateResult.statusCode}`);
        if (generateResult.data) {
            console.log(`     响应: ${JSON.stringify(generateResult.data)}`);
        }
        return false;
    }
}

// 测试AI内容分析
async function testAIContentAnalysis(token) {
    console.log('🧠 测试AI内容分析...');
    
    // 1. 检查AI引擎状态
    console.log('  🔍 检查AI引擎状态...');
    const engineOptions = createRequestOptions('/api/v1/ai-content/engine-status', 'GET');
    const engineResult = await makeRequest(engineOptions);
    
    testStats.total++;
    
    if (engineResult.success) {
        testStats.successful++;
        console.log(`  ✅ AI引擎状态正常 (${engineResult.responseTime}ms)`);
    } else {
        testStats.failed++;
        console.log(`  ❌ AI引擎状态检查失败: ${engineResult.error || engineResult.statusCode}`);
    }
    
    // 2. 测试URL分析
    console.log('  🌐 测试URL分析功能...');
    const analysisOptions = createRequestOptions('/api/v1/ai-content/url-analysis', 'POST', token);
    const analysisResult = await makeRequest(analysisOptions, TEST_DATA.urlAnalysisTest);
    
    testStats.total++;
    
    if (analysisResult.success) {
        testStats.successful++;
        console.log(`  ✅ URL分析请求成功 (${analysisResult.responseTime}ms)`);
        return true;
    } else {
        testStats.failed++;
        console.log(`  ❌ URL分析失败: ${analysisResult.error || analysisResult.statusCode}`);
        return false;
    }
}

// 清理测试资源
async function cleanupResources(token) {
    console.log('🧹 清理测试资源...');
    
    for (const resource of testStats.createdResources) {
        if (resource.type === 'course') {
            try {
                const options = createRequestOptions(`/api/v1/courses/${resource.id}`, 'DELETE', token);
                const result = await makeRequest(options);
                
                if (result.success) {
                    console.log(`  ✅ 已删除课件 ${resource.id}`);
                } else {
                    console.log(`  ⚠️ 删除课件 ${resource.id} 失败`);
                }
            } catch (error) {
                console.log(`  ⚠️ 删除课件 ${resource.id} 出错: ${error.message}`);
            }
        }
    }
}

// 生成测试报告
function generateReport() {
    const report = {
        timestamp: new Date().toISOString(),
        summary: {
            total: testStats.total,
            successful: testStats.successful,
            failed: testStats.failed,
            timeouts: testStats.timeouts,
            successRate: testStats.total > 0 ? (testStats.successful / testStats.total * 100).toFixed(1) : 0
        },
        config: CONFIG,
        testData: TEST_DATA,
        createdResources: testStats.createdResources
    };
    
    try {
        fs.writeFileSync('audio_ppt_generation_test_report.json', JSON.stringify(report, null, 2));
        console.log('\n📁 详细测试报告已保存: audio_ppt_generation_test_report.json');
    } catch (error) {
        console.log('\n⚠️ 无法保存详细报告:', error.message);
    }
}

// 主测试函数
async function runAudioPPTGenerationTests() {
    console.log('🚀 开始音频生成和DashScope PPT生成功能测试');
    console.log(`🎯 服务器: ${CONFIG.BASE_URL}`);
    console.log(`⏱️  超时时间: ${CONFIG.TIMEOUT_MS/1000} 秒`);
    console.log(`🔄 轮询间隔: ${CONFIG.POLL_INTERVAL_MS/1000} 秒`);
    console.log('');
    
    const startTime = Date.now();
    
    try {
        // 1. 登录
        const token = await login();
        console.log('');
        
        // 2. 创建测试课件
        const courseId = await createTestCourse(token);
        console.log('');
        
        // 3. 测试音频预览（不需要课件）
        await testAudioPreview(token);
        console.log('');
        
        // 4. 测试音频生成（需要课件）
        if (courseId) {
            await testAudioGeneration(courseId, token);
            console.log('');
        } else {
            console.log('⏭️ 跳过音频生成测试（课件创建失败）\n');
        }
        
        // 5. 测试DashScope PPT生成
        await testDashScopePPTGeneration(token);
        console.log('');
        
        // 6. 测试AI内容分析
        await testAIContentAnalysis(token);
        console.log('');
        
        // 7. 清理资源
        await cleanupResources(token);
        
    } catch (error) {
        console.error('❌ 测试执行出错:', error);
    }
    
    const endTime = Date.now();
    const totalTime = endTime - startTime;
    
    // 输出结果
    console.log('\n' + '='.repeat(60));
    console.log('📊 音频生成和DashScope PPT生成功能测试结果');
    console.log('='.repeat(60));
    console.log(`⏱️  总耗时: ${(totalTime/1000).toFixed(2)} 秒`);
    console.log(`📊 总测试数: ${testStats.total}`);
    console.log(`✅ 成功: ${testStats.successful} (${((testStats.successful/testStats.total)*100).toFixed(1)}%)`);
    console.log(`❌ 失败: ${testStats.failed} (${((testStats.failed/testStats.total)*100).toFixed(1)}%)`);
    console.log(`⏰ 超时: ${testStats.timeouts} (${((testStats.timeouts/testStats.total)*100).toFixed(1)}%)`);
    
    // 功能可用性评估
    console.log('\n🎯 功能可用性评估:');
    console.log('-'.repeat(40));
    const successRate = (testStats.successful/testStats.total)*100;
    
    if (successRate >= 80) {
        console.log('🎉 优秀! 音频生成和DashScope PPT生成功能基本可用');
    } else if (successRate >= 60) {
        console.log('✅ 良好! 部分功能可用，需要进一步优化');
    } else if (successRate >= 40) {
        console.log('⚠️ 一般! 功能有限，需要重点检查配置');
    } else {
        console.log('❌ 较差! 大部分功能不可用，需要检查服务配置');
    }
    
    console.log(`成功率: ${successRate.toFixed(1)}% (目标: ≥80%)`);
    
    // 生成报告
    generateReport();
}

// 运行测试
if (require.main === module) {
    runAudioPPTGenerationTests().catch(error => {
        console.error('❌ 测试执行失败:', error);
        process.exit(1);
    });
}

module.exports = { runAudioPPTGenerationTests };
