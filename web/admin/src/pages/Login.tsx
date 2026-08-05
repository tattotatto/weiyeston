import { useState } from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { setToken, setRefreshToken } from '@/utils/token';
import { useAuthStore } from '@/stores/authStore';
import { login as loginApi } from '@/api/auth';

interface LoginFormValues {
  username: string;
  password: string;
}

function Login() {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const storeLogin = useAuthStore((state) => state.login);

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true);
    try {
      const res = await loginApi(values);
      const { access_token, refresh_token, user } = res.data;

      setToken(access_token);
      setRefreshToken(refresh_token);
      storeLogin(access_token, user);

      message.success(`欢迎回来，${user.nickname || user.username}`);
      navigate('/dashboard', { replace: true });
    } catch (err: unknown) {
      const axiosError = err as { response?: { data?: { msg?: string } } };
      const msg = axiosError.response?.data?.msg || '登录失败，请重试';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <Form
        form={form}
        onFinish={handleSubmit}
        size="large"
      >
        <Form.Item
          name="username"
          rules={[{ required: true, message: '请输入用户名' }]}
        >
          <Input
            prefix={<UserOutlined />}
            placeholder="用户名"
          />
        </Form.Item>
        <Form.Item
          name="password"
          rules={[{ required: true, message: '请输入密码' }]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="密码"
          />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>
            登录
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
}

export default Login;
