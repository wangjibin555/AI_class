#!/usr/bin/env node

/**
 * 简化版性能测试脚本
 * 快速测试服务器性能，模拟20个并发客户端
 */

const https = require('https');

// 配置
const BASE_URL = 'wangjibin-sc.wepie.com';
const PORT = 9000;
const CONCURRENT_CLIENTS = 20;
const REQUESTS_PER_CLIENT = 10; // 每个客户端发送的请求数

// 测试端点（排除生成类API）
const ENDPOINTS = [
    '/api/v1/health',
    '/api/v1/test', 
    '/api/v1/courses/public',
    '/api/v1/ai-assistant/popular-questions',
    '/api/v1/ai-assistant/suggested-topics',
    '/api/v1/config/client'
];

// 执行HTTP GET请求
function makeRequest(path) {
    return new Promise((resolve) => {
        const startTime = Date.now();
        
        const options = {
            hostname: BASE_URL,
            port: PORT,
            path: path,
            method: 'GET',
            rejectUnauthorized: false
        };

        const req = https.request(options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                const responseTime = Date.now() - startTime;
                resolve({
                    path,
                    statusCode: res.statusCode,
                    responseTime,
                    success: res.statusCode >= 200 && res.statusCode < 400
                });
            });
        });

        req.on('error', (error) => {
            resolve({
                path,
                statusCode: 0,
                responseTime: Date.now() - startTime,
                success: false,
                error: error.message
            });
        });

        req.end();
    });
}

// 单个客户端测试
async function runClient(clientId) {
    console.log(`🤖 客户端${clientId} 开始测试...`);
    const results = [];
    
    for (let i = 0; i < REQUESTS_PER_CLIENT; i++) {
        const endpoint = ENDPOINTS[Math.floor(Math.random() * ENDPOINTS.length)];
        const result = await makeRequest(endpoint);
        results.push(result);
        
        // 短暂延迟
        await new Promise(resolve => setTimeout(resolve, 50 + Math.random() * 100));
    }
    
    console.log(`✅ 客户端${clientId} 完成 ${results.length} 个请求`);
    return results;
}

// 主测试函数
async function main() {
    console.log('🚀 开始性能测试');
    console.log(`🎯 服务器: https://${BASE_URL}:${PORT}`);
    console.log(`👥 并发客户端: ${CONCURRENT_CLIENTS}`);
    console.log(`📊 每客户端请求数: ${REQUESTS_PER_CLIENT}`);
    console.log(`📋 测试端点: ${ENDPOINTS.length} 个`);
    console.log('');

    const startTime = Date.now();

    // 启动所有客户端
    const clientPromises = [];
    for (let i = 1; i <= CONCURRENT_CLIENTS; i++) {
        clientPromises.push(runClient(i));
    }

    // 等待所有客户端完成
    const allResults = await Promise.all(clientPromises);
    const endTime = Date.now();
    
    // 合并所有结果
    const results = allResults.flat();
    const totalTime = endTime - startTime;
    
    // 统计
    const totalRequests = results.length;
    const successfulRequests = results.filter(r => r.success).length;
    const failedRequests = totalRequests - successfulRequests;
    const avgResponseTime = results.reduce((sum, r) => sum + r.responseTime, 0) / totalRequests;
    const qps = totalRequests / (totalTime / 1000);
    
    // 输出结果
    console.log('\n' + '='.repeat(50));
    console.log('📈 测试结果');
    console.log('='.repeat(50));
    console.log(`⏱️  总耗时: ${(totalTime/1000).toFixed(2)} 秒`);
    console.log(`📊 总请求数: ${totalRequests}`);
    console.log(`✅ 成功: ${successfulRequests} (${((successfulRequests/totalRequests)*100).toFixed(1)}%)`);
    console.log(`❌ 失败: ${failedRequests}`);
    console.log(`🚀 QPS: ${qps.toFixed(2)} 请求/秒`);
    console.log(`⚡ 平均响应时间: ${avgResponseTime.toFixed(0)} ms`);
    
    // 各端点统计
    const endpointStats = {};
    results.forEach(r => {
        if (!endpointStats[r.path]) {
            endpointStats[r.path] = { total: 0, success: 0, avgTime: 0, times: [] };
        }
        endpointStats[r.path].total++;
        if (r.success) endpointStats[r.path].success++;
        endpointStats[r.path].times.push(r.responseTime);
    });
    
    console.log('\n📋 各端点统计:');
    Object.entries(endpointStats).forEach(([path, stats]) => {
        const successRate = ((stats.success / stats.total) * 100).toFixed(1);
        const avgTime = (stats.times.reduce((a, b) => a + b, 0) / stats.times.length).toFixed(0);
        console.log(`${path}: ${stats.total}次请求, ${successRate}%成功率, ${avgTime}ms平均响应`);
    });
    
    // 性能评估
    console.log('\n🎯 性能评估:');
    if (qps >= 20 && (successfulRequests/totalRequests) >= 0.95) {
        console.log('🎉 优秀! 达到20 QPS目标且成功率>95%');
    } else if (qps >= 15) {
        console.log('✅ 良好! 接近20 QPS目标');
    } else {
        console.log('⚠️ 需要优化! 未达到20 QPS目标');
    }
}

// 运行测试
main().catch(console.error);
