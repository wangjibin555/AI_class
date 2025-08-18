#!/usr/bin/env node

/**
 * 性能测试脚本 - 模拟20个并发客户端测试API性能
 * 测试目标：除了Coze生成PPT和练习生成之外的所有端口
 * 期望结果：1分钟内处理完成（约20 QPS）
 */

const https = require('https');
const http = require('http');
const { URL } = require('url');

// 配置
const CONFIG = {
    BASE_URL: 'https://wangjibin-sc.wepie.com:9000',
    CONCURRENT_CLIENTS: 20,
    TEST_DURATION_MS: 60000, // 1分钟
    MOCK_USER_CREDENTIALS: {
        code: 'test_code_' + Date.now(),
        userInfo: {
            nickName: '性能测试用户',
            avatarUrl: 'https://example.com/avatar.png'
        }
    }
};

// 测试结果统计
const stats = {
    totalRequests: 0,
    successfulRequests: 0,
    failedRequests: 0,
    averageResponseTime: 0,
    responseTimeSum: 0,
    errors: {},
    endpointStats: {}
};

// 要测试的API端点（排除Coze生成PPT和练习生成）
const TEST_ENDPOINTS = [
    // 公开端点（无需认证）
    { method: 'GET', path: '/api/v1/health', needAuth: false, description: '健康检查' },
    { method: 'GET', path: '/api/v1/test', needAuth: false, description: '测试路由' },
    { method: 'GET', path: '/api/v1/courses/public', needAuth: false, description: '公开课件列表' },
    { method: 'GET', path: '/api/v1/ai-assistant/popular-questions', needAuth: false, description: '热门问题' },
    { method: 'GET', path: '/api/v1/ai-assistant/suggested-topics', needAuth: false, description: '推荐话题' },
    { method: 'GET', path: '/api/v1/ai-content/test', needAuth: false, description: 'AI内容测试' },
    { method: 'GET', path: '/api/v1/config/client', needAuth: false, description: '客户端配置' },
    
    // 需要认证的端点
    { method: 'POST', path: '/api/v1/auth/login', needAuth: false, description: '用户登录', 
      body: CONFIG.MOCK_USER_CREDENTIALS },
    { method: 'GET', path: '/api/v1/auth/me', needAuth: true, description: '获取用户信息' },
    { method: 'GET', path: '/api/v1/courses', needAuth: true, description: '课件列表' },
    { method: 'GET', path: '/api/v1/courses/search?keyword=test', needAuth: true, description: '搜索课件' },
    { method: 'GET', path: '/api/v1/courses/latest', needAuth: true, description: '最新课件' },
    { method: 'GET', path: '/api/v1/learning/records', needAuth: true, description: '学习记录' },
    { method: 'GET', path: '/api/v1/learning/stats', needAuth: true, description: '学习统计' },
    { method: 'GET', path: '/api/v1/ai-assistant/history', needAuth: true, description: 'AI助手历史' },
    { method: 'GET', path: '/api/v1/ai-assistant/stats', needAuth: true, description: 'AI助手统计' },
    { method: 'GET', path: '/api/v1/ai-content/analysis-history', needAuth: true, description: 'AI分析历史' },
    { method: 'GET', path: '/api/v1/ai-content/engine-status', needAuth: true, description: 'AI引擎状态' },
];

// 执行HTTP请求的函数
function makeRequest(endpoint, token = null) {
    return new Promise((resolve) => {
        const startTime = Date.now();
        const url = new URL(CONFIG.BASE_URL + endpoint.path);
        
        const options = {
            hostname: url.hostname,
            port: url.port,
            path: url.pathname + url.search,
            method: endpoint.method,
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'PerformanceTest/1.0'
            },
            rejectUnauthorized: false // 忽略SSL证书验证（仅测试用）
        };

        // 添加认证头
        if (endpoint.needAuth && token) {
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
                
                const result = {
                    endpoint: endpoint.path,
                    method: endpoint.method,
                    statusCode: res.statusCode,
                    responseTime,
                    success: res.statusCode >= 200 && res.statusCode < 400,
                    data: data,
                    error: null
                };
                
                resolve(result);
            });
        });

        req.on('error', (error) => {
            const endTime = Date.now();
            const responseTime = endTime - startTime;
            
            resolve({
                endpoint: endpoint.path,
                method: endpoint.method,
                statusCode: 0,
                responseTime,
                success: false,
                data: null,
                error: error.message
            });
        });

        // 发送请求体（如果有）
        if (endpoint.body) {
            req.write(JSON.stringify(endpoint.body));
        }
        
        req.end();
    });
}

// 登录获取token
async function login() {
    console.log('🔐 正在登录获取认证token...');
    
    const loginEndpoint = TEST_ENDPOINTS.find(e => e.path === '/api/v1/auth/login');
    const result = await makeRequest(loginEndpoint);
    
    if (result.success) {
        try {
            const responseData = JSON.parse(result.data);
            const token = responseData.data?.token || responseData.token;
            if (token) {
                console.log('✅ 登录成功，获得token');
                return token;
            } else {
                console.log('⚠️ 登录响应中未找到token，可能使用Mock模式');
                return 'mock_token_' + Date.now();
            }
        } catch (e) {
            console.log('⚠️ 解析登录响应失败，使用Mock token');
            return 'mock_token_' + Date.now();
        }
    } else {
        console.log('❌ 登录失败:', result.error || `状态码: ${result.statusCode}`);
        console.log('🔄 使用Mock token继续测试');
        return 'mock_token_' + Date.now();
    }
}

// 单个客户端测试函数
async function runClientTest(clientId, token, testDurationMs) {
    const clientStats = {
        requests: 0,
        successes: 0,
        failures: 0,
        totalResponseTime: 0
    };
    
    const startTime = Date.now();
    console.log(`🤖 客户端${clientId} 开始测试...`);
    
    while (Date.now() - startTime < testDurationMs) {
        // 随机选择一个端点进行测试
        const endpoint = TEST_ENDPOINTS[Math.floor(Math.random() * TEST_ENDPOINTS.length)];
        
        // 跳过登录端点（已经测试过）
        if (endpoint.path === '/api/v1/auth/login') {
            continue;
        }
        
        const result = await makeRequest(endpoint, token);
        
        // 更新统计
        clientStats.requests++;
        clientStats.totalResponseTime += result.responseTime;
        
        if (result.success) {
            clientStats.successes++;
        } else {
            clientStats.failures++;
            const errorKey = `${result.statusCode}_${result.error || 'Unknown'}`;
            stats.errors[errorKey] = (stats.errors[errorKey] || 0) + 1;
        }
        
        // 更新端点统计
        const endpointKey = `${result.method} ${result.endpoint}`;
        if (!stats.endpointStats[endpointKey]) {
            stats.endpointStats[endpointKey] = {
                requests: 0,
                successes: 0,
                failures: 0,
                totalResponseTime: 0,
                minResponseTime: Infinity,
                maxResponseTime: 0
            };
        }
        
        const epStats = stats.endpointStats[endpointKey];
        epStats.requests++;
        epStats.totalResponseTime += result.responseTime;
        epStats.minResponseTime = Math.min(epStats.minResponseTime, result.responseTime);
        epStats.maxResponseTime = Math.max(epStats.maxResponseTime, result.responseTime);
        
        if (result.success) {
            epStats.successes++;
        } else {
            epStats.failures++;
        }
        
        // 短暂延迟避免过于激进的请求
        await new Promise(resolve => setTimeout(resolve, Math.random() * 100));
    }
    
    console.log(`✅ 客户端${clientId} 完成测试，发送 ${clientStats.requests} 个请求`);
    return clientStats;
}

// 主测试函数
async function runPerformanceTest() {
    console.log('🚀 开始性能测试');
    console.log(`📊 目标: ${CONFIG.CONCURRENT_CLIENTS} 个并发客户端，测试时长 ${CONFIG.TEST_DURATION_MS/1000} 秒`);
    console.log(`🎯 服务器: ${CONFIG.BASE_URL}`);
    console.log(`📋 测试端点数: ${TEST_ENDPOINTS.length}`);
    console.log('');
    
    // 登录获取token
    const token = await login();
    console.log('');
    
    // 启动所有客户端
    console.log(`🏁 启动 ${CONFIG.CONCURRENT_CLIENTS} 个并发客户端...`);
    const testStartTime = Date.now();
    
    const clientPromises = [];
    for (let i = 1; i <= CONFIG.CONCURRENT_CLIENTS; i++) {
        clientPromises.push(runClientTest(i, token, CONFIG.TEST_DURATION_MS));
    }
    
    // 等待所有客户端完成
    const clientResults = await Promise.all(clientPromises);
    const testEndTime = Date.now();
    const actualTestDuration = testEndTime - testStartTime;
    
    // 汇总统计
    stats.totalRequests = clientResults.reduce((sum, client) => sum + client.requests, 0);
    stats.successfulRequests = clientResults.reduce((sum, client) => sum + client.successes, 0);
    stats.failedRequests = clientResults.reduce((sum, client) => sum + client.failures, 0);
    stats.responseTimeSum = clientResults.reduce((sum, client) => sum + client.totalResponseTime, 0);
    stats.averageResponseTime = stats.responseTimeSum / stats.totalRequests;
    
    // 输出测试结果
    console.log('\n' + '='.repeat(60));
    console.log('📈 性能测试结果');
    console.log('='.repeat(60));
    console.log(`⏱️  实际测试时长: ${(actualTestDuration/1000).toFixed(2)} 秒`);
    console.log(`📊 总请求数: ${stats.totalRequests}`);
    console.log(`✅ 成功请求: ${stats.successfulRequests} (${((stats.successfulRequests/stats.totalRequests)*100).toFixed(2)}%)`);
    console.log(`❌ 失败请求: ${stats.failedRequests} (${((stats.failedRequests/stats.totalRequests)*100).toFixed(2)}%)`);
    console.log(`🚀 QPS (每秒请求数): ${(stats.totalRequests / (actualTestDuration/1000)).toFixed(2)}`);
    console.log(`⚡ 平均响应时间: ${stats.averageResponseTime.toFixed(2)} ms`);
    
    // 显示端点详细统计
    console.log('\n📋 各端点详细统计:');
    console.log('-'.repeat(80));
    console.log('端点'.padEnd(40) + '请求数'.padEnd(8) + '成功率'.padEnd(8) + '平均响应时间'.padEnd(12) + '最小/最大');
    console.log('-'.repeat(80));
    
    for (const [endpoint, epStats] of Object.entries(stats.endpointStats)) {
        const successRate = ((epStats.successes / epStats.requests) * 100).toFixed(1);
        const avgResponseTime = (epStats.totalResponseTime / epStats.requests).toFixed(0);
        const minMax = `${epStats.minResponseTime}/${epStats.maxResponseTime}ms`;
        
        console.log(
            endpoint.padEnd(40) + 
            epStats.requests.toString().padEnd(8) + 
            `${successRate}%`.padEnd(8) + 
            `${avgResponseTime}ms`.padEnd(12) + 
            minMax
        );
    }
    
    // 显示错误统计
    if (Object.keys(stats.errors).length > 0) {
        console.log('\n❌ 错误统计:');
        console.log('-'.repeat(40));
        for (const [error, count] of Object.entries(stats.errors)) {
            console.log(`${error}: ${count} 次`);
        }
    }
    
    // 性能评估
    console.log('\n🎯 性能评估:');
    console.log('-'.repeat(40));
    const qps = stats.totalRequests / (actualTestDuration/1000);
    const successRate = (stats.successfulRequests/stats.totalRequests)*100;
    
    if (qps >= 20 && successRate >= 95) {
        console.log('🎉 优秀! 系统性能达到预期目标');
    } else if (qps >= 15 && successRate >= 90) {
        console.log('✅ 良好! 系统性能基本达到要求');
    } else if (qps >= 10 && successRate >= 80) {
        console.log('⚠️ 一般! 系统性能有待提升');
    } else {
        console.log('❌ 较差! 系统性能需要优化');
    }
    
    console.log(`目标QPS: >=20, 实际QPS: ${qps.toFixed(2)}`);
    console.log(`目标成功率: >=95%, 实际成功率: ${successRate.toFixed(2)}%`);
    console.log(`目标响应时间: <500ms, 实际平均响应时间: ${stats.averageResponseTime.toFixed(2)}ms`);
}

// 运行测试
if (require.main === module) {
    runPerformanceTest().catch(console.error);
}

module.exports = { runPerformanceTest, CONFIG };
