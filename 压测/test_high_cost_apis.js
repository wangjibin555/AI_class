#!/usr/bin/env node

/**
 * 高耗时API测试脚本
 * 专门测试音频生成和PPT生成相关的API端点
 * 这些操作通常需要较长时间，采用不同的测试策略
 */

const https = require('https');
const http = require('http');
const { URL } = require('url');
const fs = require('fs');

// 配置
const CONFIG = {
    BASE_URL: 'https://wangjibin-sc.wepie.com:9000',
    TIMEOUT_MS: 300000, // 5分钟超时
    MAX_CONCURRENT: 3,  // 最大并发数（避免服务器过载）
    POLL_INTERVAL_MS: 5000, // 状态轮询间隔
    MOCK_USER_CREDENTIALS: {
        code: 'test_code_' + Date.now(),
        userInfo: {
            nickName: '高耗时API测试用户',
            avatarUrl: 'https://example.com/avatar.png'
        }
    }
};

// 测试统计
const stats = {
    totalTests: 0,
    successfulTests: 0,
    failedTests: 0,
    timeouts: 0,
    errors: {},
    testResults: []
};

// 要测试的高耗时API端点
const HIGH_COST_ENDPOINTS = [
    // 音频生成相关
    {
        category: '音频生成',
        tests: [
            {
                name: '音频预览测试',
                method: 'POST',
                path: '/api/v1/audio/preview',
                needAuth: true,
                body: {
                    text: '这是一个音频生成测试，用于验证TTS服务是否正常工作。',
                    voice_type: 'zhixiaobai',
                    speed: 1.0,
                    volume: 50
                },
                timeout: 30000,
                description: '测试音频预览功能'
            }
        ]
    },
    
    // DashScope PPT生成相关
    {
        category: 'DashScope PPT生成',
        tests: [
            {
                name: 'DashScope引擎状态',
                method: 'GET',
                path: '/api/v1/ai-engines',
                needAuth: false,
                timeout: 10000,
                description: '检查DashScope引擎状态'
            },
            {
                name: 'PPT生成状态查询',
                method: 'GET',
                path: '/api/v1/ppt/status/test',
                needAuth: true,
                timeout: 10000,
                description: '测试PPT生成状态查询'
            }
        ]
    },
    
    // AI内容生成相关
    {
        category: 'AI内容生成',
        tests: [
            {
                name: 'AI引擎状态',
                method: 'GET',
                path: '/api/v1/ai-content/engine-status',
                needAuth: false,
                timeout: 10000,
                description: '检查AI引擎状态'
            },
            {
                name: 'AI引擎列表',
                method: 'GET',
                path: '/api/v1/ai-engines',
                needAuth: false,
                timeout: 10000,
                description: '获取可用AI引擎列表'
            }
        ]
    },
    
    // PPT相关状态查询
    {
        category: 'PPT状态查询',
        tests: [
            {
                name: '文件类型支持',
                method: 'GET',
                path: '/api/v1/content/supported-types',
                needAuth: false,
                timeout: 10000,
                description: '获取支持的文件类型'
            },
            {
                name: '幻灯片限制',
                method: 'GET',
                path: '/api/v1/content/slide-limits',
                needAuth: true,
                timeout: 10000,
                description: '获取幻灯片数量限制'
            }
        ]
    }
];

// 执行HTTP请求
function makeRequest(test, token = null) {
    return new Promise((resolve) => {
        const startTime = Date.now();
        const url = new URL(CONFIG.BASE_URL + test.path);
        
        const options = {
            hostname: url.hostname,
            port: url.port,
            path: url.pathname + url.search,
            method: test.method,
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'HighCostAPITest/1.0'
            },
            rejectUnauthorized: false,
            timeout: test.timeout || CONFIG.TIMEOUT_MS
        };

        // 添加认证头
        if (test.needAuth && token) {
            options.headers['Authorization'] = `Bearer ${token}`;
        }

        const protocol = url.protocol === 'https:' ? https : http;
        const req = protocol.request(options, (res) => {
            let data = '';
            
            res.on('data', (chunk) => {
                data += chunk;
            });
            
            res.on('end', () => {
                const endTime = Date.now();
                const responseTime = endTime - startTime;
                
                let parsedData = null;
                try {
                    parsedData = JSON.parse(data);
                } catch (e) {
                    // 忽略JSON解析错误
                }
                
                const result = {
                    test: test.name,
                    category: test.category,
                    method: test.method,
                    path: test.path,
                    statusCode: res.statusCode,
                    responseTime,
                    success: res.statusCode >= 200 && res.statusCode < 400,
                    data: parsedData,
                    rawData: data,
                    error: null,
                    description: test.description
                };
                
                resolve(result);
            });
        });

        req.on('error', (error) => {
            const endTime = Date.now();
            const responseTime = endTime - startTime;
            
            resolve({
                test: test.name,
                category: test.category,
                method: test.method,
                path: test.path,
                statusCode: 0,
                responseTime,
                success: false,
                data: null,
                rawData: null,
                error: error.message,
                description: test.description
            });
        });

        req.on('timeout', () => {
            req.destroy();
            const endTime = Date.now();
            const responseTime = endTime - startTime;
            
            resolve({
                test: test.name,
                category: test.category,
                method: test.method,
                path: test.path,
                statusCode: 0,
                responseTime,
                success: false,
                data: null,
                rawData: null,
                error: 'Request timeout',
                description: test.description
            });
        });

        // 发送请求体（如果有）
        if (test.body) {
            req.write(JSON.stringify(test.body));
        }
        
        req.end();
    });
}

// 登录获取token
async function login() {
    console.log('🔐 正在登录获取认证token...');
    
    const loginTest = {
        method: 'POST',
        path: '/api/v1/auth/login',
        body: CONFIG.MOCK_USER_CREDENTIALS,
        timeout: 10000
    };
    
    const result = await makeRequest(loginTest);
    
    if (result.success && result.data) {
        const token = result.data.data?.token || result.data.token;
        if (token) {
            console.log('✅ 登录成功');
            return token;
        }
    }
    
    console.log('⚠️ 登录失败或使用Mock模式，使用模拟token');
    return 'mock_token_' + Date.now();
}

// 运行单个测试
async function runSingleTest(test, token) {
    console.log(`  🧪 测试: ${test.name} (${test.method} ${test.path})`);
    
    const result = await makeRequest(test, token);
    
    // 更新统计
    stats.totalTests++;
    if (result.success) {
        stats.successfulTests++;
        console.log(`    ✅ 成功 (${result.responseTime}ms) - ${result.description}`);
    } else {
        stats.failedTests++;
        if (result.error === 'Request timeout') {
            stats.timeouts++;
            console.log(`    ⏰ 超时 (${result.responseTime}ms) - ${result.description}`);
        } else {
            console.log(`    ❌ 失败 (${result.statusCode}) - ${result.error || '未知错误'}`);
        }
        
        const errorKey = `${result.statusCode}_${result.error || 'Unknown'}`;
        stats.errors[errorKey] = (stats.errors[errorKey] || 0) + 1;
    }
    
    // 显示响应数据（仅前200字符）
    if (result.data) {
        const dataStr = JSON.stringify(result.data).substring(0, 200);
        console.log(`    📄 响应: ${dataStr}${dataStr.length === 200 ? '...' : ''}`);
    }
    
    stats.testResults.push(result);
    return result;
}

// 运行测试类别
async function runCategoryTests(category, token) {
    console.log(`\n🗂️  测试类别: ${category.category}`);
    console.log('─'.repeat(50));
    
    const results = [];
    
    // 串行执行测试（避免过载）
    for (const test of category.tests) {
        test.category = category.category;
        const result = await runSingleTest(test, token);
        results.push(result);
        
        // 测试间短暂延迟
        await new Promise(resolve => setTimeout(resolve, 2000));
    }
    
    return results;
}

// 主测试函数
async function runHighCostAPITests() {
    console.log('🚀 开始高耗时API测试');
    console.log(`🎯 服务器: ${CONFIG.BASE_URL}`);
    console.log(`⏱️  超时时间: ${CONFIG.TIMEOUT_MS/1000} 秒`);
    console.log(`🔄 轮询间隔: ${CONFIG.POLL_INTERVAL_MS/1000} 秒`);
    console.log('');
    
    const testStartTime = Date.now();
    
    // 登录获取token
    const token = await login();
    console.log('');
    
    // 按类别执行测试
    for (const category of HIGH_COST_ENDPOINTS) {
        await runCategoryTests(category, token);
    }
    
    const testEndTime = Date.now();
    const totalTestTime = testEndTime - testStartTime;
    
    // 输出测试结果
    console.log('\n' + '='.repeat(60));
    console.log('📊 高耗时API测试结果');
    console.log('='.repeat(60));
    console.log(`⏱️  总测试时长: ${(totalTestTime/1000).toFixed(2)} 秒`);
    console.log(`📊 总测试数: ${stats.totalTests}`);
    console.log(`✅ 成功: ${stats.successfulTests} (${((stats.successfulTests/stats.totalTests)*100).toFixed(1)}%)`);
    console.log(`❌ 失败: ${stats.failedTests} (${((stats.failedTests/stats.totalTests)*100).toFixed(1)}%)`);
    console.log(`⏰ 超时: ${stats.timeouts} (${((stats.timeouts/stats.totalTests)*100).toFixed(1)}%)`);
    
    // 按类别统计
    console.log('\n📋 分类别结果统计:');
    console.log('-'.repeat(60));
    const categoryStats = {};
    stats.testResults.forEach(result => {
        if (!categoryStats[result.category]) {
            categoryStats[result.category] = { total: 0, success: 0, failed: 0, timeouts: 0 };
        }
        categoryStats[result.category].total++;
        if (result.success) {
            categoryStats[result.category].success++;
        } else if (result.error === 'Request timeout') {
            categoryStats[result.category].timeouts++;
        } else {
            categoryStats[result.category].failed++;
        }
    });
    
    for (const [category, catStats] of Object.entries(categoryStats)) {
        const successRate = ((catStats.success / catStats.total) * 100).toFixed(1);
        console.log(`${category}: ${catStats.success}/${catStats.total} 成功 (${successRate}%)`);
        if (catStats.timeouts > 0) {
            console.log(`  ⏰ 超时: ${catStats.timeouts}`);
        }
        if (catStats.failed > 0) {
            console.log(`  ❌ 失败: ${catStats.failed}`);
        }
    }
    
    // 响应时间统计
    const responseTimes = stats.testResults
        .filter(r => r.success)
        .map(r => r.responseTime);
    
    if (responseTimes.length > 0) {
        const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
        const minResponseTime = Math.min(...responseTimes);
        const maxResponseTime = Math.max(...responseTimes);
        
        console.log('\n⚡ 响应时间统计 (仅成功请求):');
        console.log(`平均: ${avgResponseTime.toFixed(0)}ms`);
        console.log(`最快: ${minResponseTime}ms`);
        console.log(`最慢: ${maxResponseTime}ms`);
    }
    
    // 错误统计
    if (Object.keys(stats.errors).length > 0) {
        console.log('\n❌ 错误统计:');
        console.log('-'.repeat(40));
        for (const [error, count] of Object.entries(stats.errors)) {
            console.log(`${error}: ${count} 次`);
        }
    }
    
    // 服务可用性评估
    console.log('\n🎯 服务可用性评估:');
    console.log('-'.repeat(40));
    const successRate = (stats.successfulTests/stats.totalTests)*100;
    const timeoutRate = (stats.timeouts/stats.totalTests)*100;
    
    if (successRate >= 90 && timeoutRate < 10) {
        console.log('🎉 优秀! 高耗时API服务稳定可用');
    } else if (successRate >= 80 && timeoutRate < 20) {
        console.log('✅ 良好! API服务基本可用');
    } else if (successRate >= 70) {
        console.log('⚠️ 一般! API服务有待改进');
    } else {
        console.log('❌ 较差! API服务需要优化');
    }
    
    console.log(`成功率: ${successRate.toFixed(1)}% (目标: ≥90%)`);
    console.log(`超时率: ${timeoutRate.toFixed(1)}% (目标: <10%)`);
    
    // 生成详细报告文件
    const reportData = {
        timestamp: new Date().toISOString(),
        config: CONFIG,
        summary: {
            totalTests: stats.totalTests,
            successfulTests: stats.successfulTests,
            failedTests: stats.failedTests,
            timeouts: stats.timeouts,
            totalTestTime: totalTestTime,
            successRate: successRate,
            timeoutRate: timeoutRate
        },
        categoryStats,
        responseTimes: responseTimes.length > 0 ? {
            average: responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length,
            min: Math.min(...responseTimes),
            max: Math.max(...responseTimes)
        } : null,
        errors: stats.errors,
        detailedResults: stats.testResults
    };
    
    try {
        fs.writeFileSync('high_cost_api_test_report.json', JSON.stringify(reportData, null, 2));
        console.log('\n📁 详细测试报告已保存: high_cost_api_test_report.json');
    } catch (error) {
        console.log('\n⚠️ 无法保存详细报告:', error.message);
    }
}

// 运行测试
if (require.main === module) {
    runHighCostAPITests().catch(error => {
        console.error('❌ 测试执行失败:', error);
        process.exit(1);
    });
}

module.exports = { runHighCostAPITests, CONFIG };
