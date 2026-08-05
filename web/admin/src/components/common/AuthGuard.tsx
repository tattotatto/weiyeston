import { useEffect, useState } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
import { Spin } from 'antd';
import { useAuthStore } from '@/stores/authStore';
import { setNavigate } from '@/utils/navigate';
import { getToken, getRefreshToken, setToken, setRefreshToken, removeToken } from '@/utils/token';
import { refreshToken as refreshTokenApi } from '@/api/auth';

interface AuthGuardProps {
  children: React.ReactNode;
}

function AuthGuard({ children }: AuthGuardProps) {
  const navigate = useNavigate();
  const { isAuthenticated, logout } = useAuthStore();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    setNavigate((path: string) => navigate(path, { replace: true }));

    const checkAuth = async () => {
      const token = getToken();
      if (!token) {
        logout();
        setChecking(false);
        return;
      }

      // 简单解析 JWT payload 检查是否过期（本地预检，不涉及验签）
      try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        if (payload.exp * 1000 < Date.now()) {
          // token 已过期，尝试刷新
          const refresh = getRefreshToken();
          if (refresh) {
            try {
              const res = await refreshTokenApi(refresh);
              const { access_token, refresh_token } = res.data;
              setToken(access_token);
              setRefreshToken(refresh_token);
            } catch {
              // 刷新失败，跳转登录
              logout();
              removeToken();
            }
          } else {
            logout();
            removeToken();
          }
        }
      } catch {
        // 解析失败，继续（由服务端中间件最终校验）
      }
      setChecking(false);
    };

    checkAuth();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  if (checking) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

export default AuthGuard;
