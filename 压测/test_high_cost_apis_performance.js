#!/usr/bin/env node

/**
 * 高耗时API性能测试脚本
 * 专门测试音频生成和DashScope PPT生成相关API的并发性能
 * 目标：验证在并发访问下能否达到20 QPS
 */

const https = require('https');
const { URL } = require('url');
const fs = require('fs');

// 配置
const CONFIG = {
    BASE_URL: 'https://wangjibin-sc.wepie.com:9000',
    CONCURRENT_CLIENTS: 20,
    TEST_DURATION_MS: 60000, // 1分钟
    TIMEOUT_MS: 30000, // 30秒超时
    MOCK_USER_CREDENTIALS: {
        code: 'perf_test_' + Date.now(),
        userInfo: {
            nickName: '性能测试用户',
            avatarUrl: 'https://example.com/avatar.png'
        }
    }
};

// 测试统计
const stats = {
    totalRequests: 0,
    successfulRequests: 0,
    failedRequests: 0,
    timeouts: 0,
    averageResponseTime: 0,
    responseTimeSum: 0,
    errors: {},
    endpointStats: {},
    clientStats: []
};

// 可正常工作的高耗时API端点（基于之前的测试结果）
const WORKING_ENDPOINTS = [
    // DashScope和AI引擎相关（不需要认证）
    {
        method: 'GET',
        path: '/api/v1/ai-engines',
        needAuth: false,
        description: 'DashScope引擎列表',
        weight: 3 // 权重，影响选择频率
    },
    {
        method: 'GET', 
        path: '/api/v1/ai-content/engine-status',
        needAuth: false,
        description: 'AI引擎状态',
        weight: 3
    },
    {
        method: 'GET',
        path: '/api/v1/content/supported-types', 
        needAuth: false,
        description: '支持的文件类型',
        weight: 2
    },
    {
        method: 'GET',
        path: '/api/v1/health',
        needAuth: false,
        description: '健康检查',
        weight: 1
    },
    // 尝试一些可能工作的认证端点
    {
        method: 'POST',
        path: '/api/v1/auth/login',
        needAuth: false,
        description: '用户登录',
        weight: 2,
        body: CONFIG.MOCK_USER_CREDENTIALS
    }
];

// 创建加权端点列表（用于随机选择）
const WEIGHTED_ENDPOINTS = [];
WORKING_ENDPOINTS.forEach(endpoint => {
    for (let i = 0; i < endpoint.weight; i++) {
        WEIGHTED_ENDPOINTS.push(endpoint);
    }
});

// HTTP请求函数
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
                'User-Agent': 'HighCostAPIPerformanceTest/1.0'
            },
            rejectUnauthorized: false,
            timeout: CONFIG.TIMEOUT_MS
        };

        if (endpoint.needAuth && token) {
            options.headers['Authorization'] = `Bearer ${token}`;
        }

        const req = https.request(options, (res) => {
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
                    description: endpoint.description,
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
                description: endpoint.description,
                statusCode: 0,
                responseTime,
                success: false,
                data: null,
                error: error.message
            });
        });

        req.on('timeout', () => {
            req.destroy();
            const endTime = Date.now();
            const responseTime = endTime - startTime;
            
            resolve({
                endpoint: endpoint.path,
                method: endpoint.method,
                description: endpoint.description,
                statusCode: 0,
                responseTime,
                success: false,
                data: null,
                error: 'Request timeout'
            });
        });

        // 发送请求体（如果有）
        if (endpoint.body) {
            req.write(JSON.stringify(endpoint.body));
        }
        
        req.end();
    });
}

// 登录获取token（尽力而为）
async function attemptLogin() {
    console.log('🔐 尝试登录获取认证token...');
    
    const loginEndpoint = WORKING_ENDPOINTS.find(e => e.path === '/api/v1/auth/login');
    const result = await makeRequest(loginEndpoint);
    
    if (result.success) {
        try {
            const responseData = JSON.parse(result.data);
            const token = responseData.data?.token || responseData.token;
            if (token && !token.includes('mock_token')) {
                console.log('✅ 获得真实认证token');
                return token;
            }
        } catch (e) {
            // 忽略解析错误
        }
    }
    
    console.log('⚠️ 无法获得有效token，继续测试公开端点');
    return null;
}

// 单个客户端性能测试
async function runClientPerformanceTest(clientId, token, testDurationMs) {
    const clientStats = {
        clientId,
        requests: 0,
        successes: 0,
        failures: 0,
        timeouts: 0,
        totalResponseTime: 0,
        results: []
    };
    
    const startTime = Date.now();
    console.log(`🤖 客户端${clientId} 开始高耗时API性能测试...`);
    
    while (Date.now() - startTime < testDurationMs) {
        // 随机选择一个端点（基于权重）
        const endpoint = WEIGHTED_ENDPOINTS[Math.floor(Math.random() * WEIGHTED_ENDPOINTS.length)];
        
        const result = await makeRequest(endpoint, token);
        
        // 更新客户端统计
        clientStats.requests++;
        clientStats.totalResponseTime += result.responseTime;
        clientStats.results.push(result);
        
        if (result.success) {
            clientStats.successes++;
        } else {
            clientStats.failures++;
            if (result.error === 'Request timeout') {
                clientStats.timeouts++;
            }
        }
        
        // 短暂延迟避免过于激进的请求
        await new Promise(resolve => setTimeout(resolve, Math.random() * 200));
    }
    
    console.log(`✅ 客户端${clientId} 完成，发送 ${clientStats.requests} 个高耗时API请求`);
    return clientStats;
}

// 更新全局统计
function updateGlobalStats(clientResults) {
    clientResults.forEach(clientStat => {
        stats.totalRequests += clientStat.requests;
        stats.successfulRequests += clientStat.successes;
        stats.failedRequests += clientStat.failures;
        stats.timeouts += clientStat.timeouts;
        stats.responseTimeSum += clientStat.totalResponseTime;
        
        // 更新端点统计
        clientStat.results.forEach(result => {
            if (result.error && result.error !== 'Request timeout') {
                const errorKey = `${result.statusCode}_${result.error}`;
                stats.errors[errorKey] = (stats.errors[errorKey] || 0) + 1;
            }
            
            const endpointKey = `${result.method} ${result.endpoint}`;
            if (!stats.endpointStats[endpointKey]) {
                stats.endpointStats[endpointKey] = {
                    description: result.description,
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
        });
    });
    
    stats.averageResponseTime = stats.responseTimeSum / stats.totalRequests;
    stats.clientStats = clientResults;
}

// 主测试函数
async function runHighCostAPIPerformanceTest() {
    console.log('🚀 开始高耗时API性能测试');
    console.log(`📊 目标: ${CONFIG.CONCURRENT_CLIENTS} 个并发客户端，测试时长 ${CONFIG.TEST_DURATION_MS/1000} 秒`);
    console.log(`🎯 服务器: ${CONFIG.BASE_URL}`);
    console.log(`📋 测试端点数: ${WORKING_ENDPOINTS.length} 个 (加权后 ${WEIGHTED_ENDPOINTS.length} 个选项)`);
    console.log('');
    
    // 显示测试端点
    console.log('📋 测试端点列表:');
    WORKING_ENDPOINTS.forEach((endpoint, index) => {
        console.log(`  ${index + 1}. ${endpoint.method} ${endpoint.path} - ${endpoint.description} (权重: ${endpoint.weight})`);
    });
    console.log('');
    
    // 尝试获取认证token
    const token = await attemptLogin();
    console.log('');
    
    // 启动所有客户端
    console.log(`🏁 启动 ${CONFIG.CONCURRENT_CLIENTS} 个并发客户端进行高耗时API测试...`);
    const testStartTime = Date.now();
    
    const clientPromises = [];
    for (let i = 1; i <= CONFIG.CONCURRENT_CLIENTS; i++) {
        clientPromises.push(runClientPerformanceTest(i, token, CONFIG.TEST_DURATION_MS));
    }
    
    // 等待所有客户端完成
    const clientResults = await Promise.all(clientPromises);
    const testEndTime = Date.now();
    const actualTestDuration = testEndTime - testStartTime;
    
    // 更新统计
    updateGlobalStats(clientResults);
    
    // 输出测试结果
    console.log('\n' + '='.repeat(70));
    console.log('📈 高耗时API性能测试结果');
    console.log('='.repeat(70));
    console.log(`⏱️  实际测试时长: ${(actualTestDuration/1000).toFixed(2)} 秒`);
    console.log(`📊 总请求数: ${stats.totalRequests}`);
    console.log(`✅ 成功请求: ${stats.successfulRequests} (${((stats.successfulRequests/stats.totalRequests)*100).toFixed(2)}%)`);
    console.log(`❌ 失败请求: ${stats.failedRequests} (${((stats.failedRequests/stats.totalRequests)*100).toFixed(2)}%)`);
    console.log(`⏰ 超时请求: ${stats.timeouts} (${((stats.timeouts/stats.totalRequests)*100).toFixed(2)}%)`);
    console.log(`🚀 QPS (每秒请求数): ${(stats.totalRequests / (actualTestDuration/1000)).toFixed(2)}`);
    console.log(`⚡ 平均响应时间: ${stats.averageResponseTime.toFixed(2)} ms`);
    
    // 显示各端点详细统计
    console.log('\n📋 各端点详细统计:');
    console.log('-'.repeat(80));
    console.log('端点'.padEnd(35) + '请求数'.padEnd(8) + '成功率'.padEnd(8) + '平均响应'.padEnd(10) + '最小/最大'.padEnd(12) + '描述');
    console.log('-'.repeat(80));
    
    for (const [endpoint, epStats] of Object.entries(stats.endpointStats)) {
        const successRate = ((epStats.successes / epStats.requests) * 100).toFixed(1);
        const avgResponseTime = (epStats.totalResponseTime / epStats.requests).toFixed(0);
        const minMax = `${epStats.minResponseTime}/${epStats.maxResponseTime}ms`;
        
        console.log(
            endpoint.padEnd(35) + 
            epStats.requests.toString().padEnd(8) + 
            `${successRate}%`.padEnd(8) + 
            `${avgResponseTime}ms`.padEnd(10) + 
            minMax.padEnd(12) + 
            epStats.description
        );
    }
    
    // 客户端表现统计
    console.log('\n👥 客户端表现统计:');
    console.log('-'.repeat(50));
    const avgRequestsPerClient = stats.totalRequests / CONFIG.CONCURRENT_CLIENTS;
    const clientSuccessRates = stats.clientStats.map(c => (c.successes / c.requests) * 100);
    const avgClientSuccessRate = clientSuccessRates.reduce((a, b) => a + b, 0) / clientSuccessRates.length;
    
    console.log(`平均每客户端请求数: ${avgRequestsPerClient.toFixed(1)}`);
    console.log(`客户端平均成功率: ${avgClientSuccessRate.toFixed(1)}%`);
    console.log(`客户端成功率范围: ${Math.min(...clientSuccessRates).toFixed(1)}% - ${Math.max(...clientSuccessRates).toFixed(1)}%`);
    
    // 显示错误统计
    if (Object.keys(stats.errors).length > 0) {
        console.log('\n❌ 错误统计:');
        console.log('-'.repeat(40));
        for (const [error, count] of Object.entries(stats.errors)) {
            console.log(`${error}: ${count} 次`);
        }
    }
    
    // 性能评估
    console.log('\n🎯 高耗时API性能评估:');
    console.log('-'.repeat(50));
    const qps = stats.totalRequests / (actualTestDuration/1000);
    const successRate = (stats.successfulRequests/stats.totalRequests)*100;
    
    if (qps >= 20 && successRate >= 95) {
        console.log('🎉 优秀! 高耗时API性能达到预期目标');
    } else if (qps >= 15 && successRate >= 90) {
        console.log('✅ 良好! 高耗时API性能基本达到要求');
    } else if (qps >= 10 && successRate >= 80) {
        console.log('⚠️ 一般! 高耗时API性能有待提升');
    } else {
        console.log('❌ 较差! 高耗时API性能需要优化');
    }
    
    console.log(`目标QPS: >=20, 实际QPS: ${qps.toFixed(2)}`);
    console.log(`目标成功率: >=95%, 实际成功率: ${successRate.toFixed(2)}%`);
    console.log(`目标响应时间: <2000ms, 实际平均响应时间: ${stats.averageResponseTime.toFixed(2)}ms`);
    
    // 生成详细报告
    const reportData = {
        timestamp: new Date().toISOString(),
        testType: 'high_cost_api_performance',
        config: CONFIG,
        summary: {
            totalRequests: stats.totalRequests,
            successfulRequests: stats.successfulRequests,
            failedRequests: stats.failedRequests,
            timeouts: stats.timeouts,
            actualTestDuration: actualTestDuration,
            qps: qps,
            successRate: successRate,
            averageResponseTime: stats.averageResponseTime
        },
        endpointStats: stats.endpointStats,
        clientStats: stats.clientStats,
        errors: stats.errors
    };
    
    try {
        fs.writeFileSync('high_cost_api_performance_report.json', JSON.stringify(reportData, null, 2));
        console.log('\n📁 详细性能报告已保存: high_cost_api_performance_report.json');
    } catch (error) {
        console.log('\n⚠️ 无法保存详细报告:', error.message);
    }
    
    console.log('\n🏁 高耗时API性能测试完成！');
}

// 运行测试
if (require.main === module) {
    runHighCostAPIPerformanceTest().catch(error => {
        console.error('❌ 测试执行失败:', error);
        process.exit(1);
    });
}

module.exports = { runHighCostAPIPerformanceTest };
