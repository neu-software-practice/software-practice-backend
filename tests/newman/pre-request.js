// ============================================================================
// NEUHIS Agent Newman Pre-request Script
// 集合级 Pre-request: 自动认证维护
// 在 Postman Collection 的 Pre-request Script 中使用
// ============================================================================

// 检查当前 token 是否有效（至少还有 60 秒有效期）
function isTokenValid() {
    var token = pm.environment.get('accessToken');
    if (!token) return false;

    try {
        // Newman 运行在 Node.js 环境下，使用 Buffer 解码 base64
        var payload = JSON.parse(Buffer.from(token.split('.')[1], 'base64').toString());
        // 检查 token 是否即将过期（60 秒缓冲）
        return payload.exp * 1000 > Date.now() + 60000;
    } catch (e) {
        console.log('Token parse error:', e.message);
        return false;
    }
}

// 获取请求路径
var requestPath = pm.request.url.toString();

// 公开端点列表 (不需要认证)
var publicEndpoints = [
    '/api/health',
    '/api/patients/verify',
    '/api/auth/register',
    '/api/auth/login',
    '/api/auth/refresh',
    '/api/auth/logout'
];

var isPublic = publicEndpoints.some(function(path) {
    return requestPath.indexOf(path) !== -1;
});

// Auth 端点自行管理 token
if (requestPath.indexOf('/api/auth/') !== -1) {
    return;
}

// 公开端点跳过认证
if (isPublic && pm.request.method === 'GET' && requestPath.indexOf('/api/health') !== -1) {
    return;
}

// 需要认证的端点：确保有有效 token
if (!isTokenValid()) {
    console.log('Token invalid or expiring soon, auto-login...');

    var baseUrl = pm.environment.get('baseUrl');
    var loginPhone = pm.environment.get('testPhone') || '13800000001';
    var loginPassword = pm.environment.get('testPassword') || 'TestPass123!';

    // 先尝试登录
    pm.sendRequest({
        url: baseUrl + '/api/auth/login',
        method: 'POST',
        header: { 'Content-Type': 'application/json' },
        body: {
            mode: 'raw',
            raw: JSON.stringify({
                phone: loginPhone,
                password: loginPassword
            })
        }
    }, function (err, res) {
        if (err) {
            console.error('Auto-login error:', err);
            return;
        }

        if (res.code === 200) {
            var data = res.json().data;
            pm.environment.set('accessToken', data.accessToken);
            pm.environment.set('refreshTokenValue', data.refreshToken);
            if (data.user) {
                pm.environment.set('patientId', data.user.patientId);
            }
            console.log('Auto-login successful');
            return;
        }

        // 登录失败则尝试注册
        console.log('Login failed (status ' + res.code + '), trying auto-register...');
        pm.sendRequest({
            url: baseUrl + '/api/auth/register',
            method: 'POST',
            header: { 'Content-Type': 'application/json' },
            body: {
                mode: 'raw',
                raw: JSON.stringify({
                    phone: loginPhone,
                    password: loginPassword,
                    realName: '黑盒测试用户'
                })
            }
        }, function (err2, res2) {
            if (err2) {
                console.error('Auto-register error:', err2);
                return;
            }
            if (res2.code === 201 || res2.code === 200) {
                var data2 = res2.json().data;
                pm.environment.set('accessToken', data2.accessToken);
                pm.environment.set('refreshTokenValue', data2.refreshToken);
                if (data2.user) {
                    pm.environment.set('patientId', data2.user.patientId);
                }
                console.log('Auto-register successful');
            } else {
                console.log('Auto-register response:', res2.code, res2.text());
            }
        });
    });
}
