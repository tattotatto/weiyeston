import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { getToken, setToken, setRefreshToken, getRefreshToken, removeToken } from '@/utils/token';
import { navigateTo } from '@/utils/navigate';

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
});

// 请求拦截器：自动附加 JWT Token
client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getToken();
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Token 刷新状态管理
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else if (token) {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

// 响应拦截器：401 自动刷新 Token
client.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<{ msg: string }>) => {
    const originalRequest = error.config;

    // 如果是刷新 token 的请求本身失败，直接跳转登录（避免死循环）
    if (originalRequest?.url === '/auth/refresh') {
      removeToken();
      navigateTo('/login');
      return Promise.reject(error);
    }

    if (error.response?.status === 401 && originalRequest && !(originalRequest as unknown as Record<string, unknown>)._retry) {
      if (isRefreshing) {
        // 正在刷新中，排队等待
        return new Promise<string>((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return client(originalRequest);
        });
      }

      (originalRequest as unknown as Record<string, unknown>)._retry = true;
      isRefreshing = true;

      try {
        const refreshToken = getRefreshToken();
        const res = await axios.post('/api/v1/auth/refresh', {
          refresh_token: refreshToken,
        }, {
          headers: { Authorization: `Bearer ${getToken()}` },
        });

        const { access_token, refresh_token } = res.data.data;
        setToken(access_token);
        setRefreshToken(refresh_token);

        processQueue(null, access_token);

        originalRequest.headers.Authorization = `Bearer ${access_token}`;
        return client(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError, null);
        removeToken();
        navigateTo('/login');
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    const msg = error.response?.data?.msg || error.message || '请求失败';
    console.error(msg);
    return Promise.reject(error);
  },
);

export default client;
