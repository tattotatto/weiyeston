import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Form, Input, Button, Checkbox, Typography, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { login as loginApi } from '@/api/auth';
import { useAuthStore } from '@/stores/authStore';
import { setToken, setRefreshToken } from '@/utils/token';

const { Text } = Typography;

function Login() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const storeLogin = useAuthStore((s) => s.login);

  const handleSubmit = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await loginApi(values);
      const { access_token, refresh_token, user } = res.data;
      setToken(access_token);
      if (refresh_token) setRefreshToken(refresh_token);
      storeLogin(access_token, user);
      message.success(`欢迎回来，${user.nickname || user.username}`);
      navigate('/dashboard', { replace: true });
    } catch {
      message.error('用户名或密码错误');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 4, textAlign: 'center' }}>
        欢迎登录
      </h2>
      <Text type="secondary" style={{ display: 'block', textAlign: 'center', marginBottom: 24, fontSize: 13 }}>
        请输入您的账号密码
      </Text>

      <Form form={form} onFinish={handleSubmit} size="large" autoComplete="off">
        <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
          <Input prefix={<UserOutlined style={{ color: '#bfbfbf' }} />} placeholder="用户名" />
        </Form.Item>

        <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
          <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="密码" />
        </Form.Item>

        <Form.Item style={{ marginBottom: 12 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Form.Item name="remember" valuePropName="checked" noStyle>
              <Checkbox style={{ fontSize: 13 }}>记住密码</Checkbox>
            </Form.Item>
            <Link to="/register" style={{ fontSize: 13 }}>还没有账号？立即注册</Link>
          </div>
        </Form.Item>

        <Form.Item style={{ marginBottom: 0 }}>
          <Button type="primary" htmlType="submit" loading={loading} block>
            立即登录
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
}

export default Login;
