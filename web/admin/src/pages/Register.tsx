import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Form, Input, Button, Typography, message } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, SmileOutlined, PhoneOutlined } from '@ant-design/icons';
import { register } from '@/api/auth';

const { Text } = Typography;

function Register() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const [form] = Form.useForm();

  const handleSubmit = async (values: {
    username: string; password: string; email: string; phone: string; nickname?: string;
  }) => {
    setLoading(true);
    try {
      await register({
        username: values.username,
        password: values.password,
        email: values.email,
        phone: values.phone,
        nickname: values.nickname || undefined,
      });
      message.success('注册成功，请等待管理员审核后登录');
      navigate('/login', { replace: true });
    } catch (err: unknown) {
      const e = err as { response?: { data?: { msg?: string } } };
      message.error(e.response?.data?.msg || '注册失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 4, textAlign: 'center' }}>
        创建账号
      </h2>
      <Text type="secondary" style={{ display: 'block', textAlign: 'center', marginBottom: 24, fontSize: 13 }}>
        注册后需等待管理员审核
      </Text>

      <Form form={form} onFinish={handleSubmit} size="large" autoComplete="off">
        <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
          <Input prefix={<UserOutlined style={{ color: '#bfbfbf' }} />} placeholder="用户名" />
        </Form.Item>

        <Form.Item name="phone" rules={[
          { required: true, message: '请输入手机号' },
          { pattern: /^1\d{10}$/, message: '手机号格式不正确' },
        ]}>
          <Input prefix={<PhoneOutlined style={{ color: '#bfbfbf' }} />} placeholder="手机号" />
        </Form.Item>

        <Form.Item name="email" rules={[
          { required: true, message: '请输入邮箱' },
          { type: 'email', message: '邮箱格式不正确' },
        ]}>
          <Input prefix={<MailOutlined style={{ color: '#bfbfbf' }} />} placeholder="邮箱" />
        </Form.Item>

        <Form.Item name="password" rules={[
          { required: true, message: '请输入密码' },
          { min: 8, message: '密码至少 8 个字符' },
        ]}>
          <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="密码（至少 8 个字符）" />
        </Form.Item>

        <Form.Item name="confirmPassword" dependencies={['password']} rules={[
          { required: true, message: '请确认密码' },
          ({ getFieldValue }) => ({
            validator(_, value) {
              if (!value || getFieldValue('password') === value) return Promise.resolve();
              return Promise.reject(new Error('两次输入的密码不一致'));
            },
          }),
        ]}>
          <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="确认密码" />
        </Form.Item>

        <Form.Item name="nickname" rules={[{ max: 50, message: '昵称不能超过 50 个字符' }]}>
          <Input prefix={<SmileOutlined style={{ color: '#bfbfbf' }} />} placeholder="昵称（可选）" />
        </Form.Item>

        <Form.Item style={{ marginBottom: 0 }}>
          <Button type="primary" htmlType="submit" loading={loading} block>
            注册
          </Button>
        </Form.Item>

        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Text style={{ fontSize: 13 }}>
            已有账号？<Link to="/login">返回登录</Link>
          </Text>
        </div>
      </Form>
    </div>
  );
}

export default Register;
