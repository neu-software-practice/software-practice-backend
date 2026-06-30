// ============================================================================
// NEUHIS Agent Admin Newman Pre-request Script
// 集合级 Pre-request: 自动 admin 认证维护
// 在 Admin Postman Collection 的 Pre-request Script 中使用
// ============================================================================

// 检查当前 admin token 是否有效（至少还有 60 秒有效期）
function isAdminTokenValid() {
    var token = pm.environment.get('adminAccessToken');
    if (!token) return false;

    try {
        // Newman 运行在 Node.js 环境下，使用 Buffer 解码 base64
        var payload = JSON.parse(Buffer.from(token.split('.')[1], 'base64').toString());
        // 检查 token 是否即将过期（60 秒缓冲）
        return payload.exp * 1000 > Date.now() + 60000;
    } catch (e) {
        console.log('Admin token parse error:', e.message);
        return false;
    }
}

// 获取请求路径
var requestPath = pm.request.url.toString();

// Admin auth 端点自行管理 token（login/logout/refresh 不需要自动认证）
if (requestPath.indexOf('/admin/auth/') !== -1) {
    return;
}

// 公开端点跳过认证
if (requestPath.indexOf('/api/health') !== -1) {
    return;
}

// 需要认证的 admin 端点：确保有有效 admin token
if (!isAdminTokenValid()) {
    console.log('Admin token invalid or expiring soon, auto-login...');

    var baseUrl = pm.environment.get('baseUrl');
    var adminUsername = pm.environment.get('adminUsername') || 'admin';
    var adminPassword = pm.environment.get('adminPassword') || 'admin123';

    pm.sendRequest({
        url: baseUrl + '/admin/auth/login',
        method: 'POST',
        header: { 'Content-Type': 'application/json' },
        body: {
            mode: 'raw',
            raw: JSON.stringify({
                username: adminUsername,
                password: adminPassword
            })
        }
    }, function (err, res) {
        if (err) {
            console.error('Admin auto-login error:', err);
            return;
        }

        if (res.code === 200) {
            var body = res.json();
            var data = body.data;
            if (data && data.tokens) {
                pm.environment.set('adminAccessToken', data.tokens.accessToken);
                pm.environment.set('adminRefreshToken', data.tokens.refreshToken);
            }
            if (data && data.user) {
                pm.environment.set('adminId', data.user.id);
            }
            console.log('Admin auto-login successful');
            return;
        }

        console.log('Admin auto-login failed (status ' + res.code + '):', res.text());
    });
}
