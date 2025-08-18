// 清除认证状态的脚本
// 在微信开发者工具控制台中运行

console.log('清除认证状态...');

// 清除所有认证相关的本地存储
wx.removeStorageSync('access_token');
wx.removeStorageSync('refresh_token');
wx.removeStorageSync('user_info');
wx.removeStorageSync('token_expires');

console.log('认证状态已清除');

// 跳转到登录页
wx.redirectTo({
  url: '/pages/login/login'
});

console.log('已跳转到登录页'); 