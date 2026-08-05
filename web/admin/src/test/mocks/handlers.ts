// web/admin/src/test/mocks/handlers.ts
// MSW (Mock Service Worker) 请求处理器

import { http, HttpResponse } from 'msw';

const BASE_URL = '/api/v1';

export const handlers = [
  // Mock 健康检查
  http.get(`${BASE_URL}/health`, () => {
    return HttpResponse.json({
      status: 'healthy',
      version: '0.1.0',
      checks: { database: { status: 'ok' }, redis: { status: 'ok' } },
    });
  }),

  // Mock 登录
  http.post(`${BASE_URL}/auth/login`, async ({ request }) => {
    const body = await request.json() as { username: string; password: string };
    if (body.username === 'admin' && body.password === 'password') {
      return HttpResponse.json({
        code: 0,
        msg: 'ok',
        data: { token: 'mock-jwt-token', expires_at: '', user: { id: 1, username: 'admin', nickname: '管理员' } },
      });
    }
    return HttpResponse.json({ code: 10001, msg: '用户名或密码错误' }, { status: 401 });
  }),

  // Mock 获取当前用户
  http.get(`${BASE_URL}/auth/me`, () => {
    return HttpResponse.json({
      code: 0,
      msg: 'ok',
      data: { id: 1, username: 'admin', nickname: '管理员' },
    });
  }),
];
