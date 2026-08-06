import { useState } from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, SmileOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { register } from '@/api/auth';

interface RegisterFormValues {
  username: string;
  password: string;
  confirmPassword: string;
  email: string;
  nickname?: string;
}

function Register() {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: RegisterFormValues) => {
    setLoading(true);
    try {
      await register({
        username: values.username,
        password: values.password,
        email: values.email,
        nickname: values.nickname || undefined,
      });
      message.success('注册成功，请等待管理员审核');
      navigate('/login', { replace: true });
    } catch (err: unknown) {
      const axiosError = err as { response?: { data?: { msg?: string } } };
      const msg = axiosError.response?.data?.msg || '注册失败，请重试';
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
          rules={[
            { required: true, message: '请输入密码' },
            { min: 8, message: '密码至少 8 个字符' },
          ]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="密码（至少 8 个字符）"
          />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          dependencies={['password']}
          rules={[
            { required: true, message: '请确认密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('password') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('两次输入的密码不一致'));
              },
            }),
          ]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="确认密码"
          />
        </Form.Item>
        <Form.Item
          name="email"
          rules={[
            { required: true, message: '请输入邮箱' },
            { type: 'email', message: '邮箱格式不正确' },
          ]}
        >
          <Input
            prefix={<MailOutlined />}
            placeholder="邮箱"
          />
        </Form.Item>
        <Form.Item
          name="nickname"
          rules={[{ max: 50, message: '昵称不能超过 50 个字符' }]}
        >
          <Input
            prefix={<SmileOutlined />}
            placeholder="昵称（可选）"
          />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>
            注册
          </Button>
        </Form.Item>
        <div style={{ textAlign: 'center' }}>
          已有账号？<Link to="/login">返回登录</Link>
        </div>
      </Form>
    </Card>
  );
}

export default Register;
