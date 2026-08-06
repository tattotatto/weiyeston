import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Form, Input, Button, Typography, message, Steps } from 'antd';
import { PhoneOutlined, LockOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { forgotPassword, resetPassword } from '@/api/auth';

const { Text } = Typography;

function ForgotPassword() {
  const [step, setStep] = useState(0); // 0=输入手机号, 1=输入验证码+新密码
  const [phone, setPhone] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const [form] = Form.useForm();

  // 第一步：发送验证码
  const handleSendCode = async (values: { phone: string }) => {
    setLoading(true);
    try {
      await forgotPassword({ phone: values.phone });
      setPhone(values.phone);
      message.success('验证码已发送（预设验证码：888888）');
      setStep(1);
      form.resetFields();
    } catch (err: unknown) {
      const e = err as { response?: { data?: { msg?: string } } };
      message.error(e.response?.data?.msg || '发送验证码失败');
    } finally {
      setLoading(false);
    }
  };

  // 第二步：重置密码
  const handleReset = async (values: { code: string; new_password: string }) => {
    setLoading(true);
    try {
      await resetPassword({
        phone,
        code: values.code,
        new_password: values.new_password,
      });
      message.success('密码重置成功，请使用新密码登录');
      navigate('/login', { replace: true });
    } catch (err: unknown) {
      const e = err as { response?: { data?: { msg?: string } } };
      message.error(e.response?.data?.msg || '密码重置失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 4, textAlign: 'center' }}>
        找回密码
      </h2>
      <Text type="secondary" style={{ display: 'block', textAlign: 'center', marginBottom: 24, fontSize: 13 }}>
        {step === 0 ? '请输入注册时使用的手机号' : '请输入验证码和新密码'}
      </Text>

      <Steps
        current={step}
        size="small"
        style={{ marginBottom: 24 }}
        items={[
          { title: '验证身份' },
          { title: '重置密码' },
        ]}
      />

      {step === 0 && (
        <Form form={form} onFinish={handleSendCode} size="large" autoComplete="off">
          <Form.Item name="phone" rules={[
            { required: true, message: '请输入手机号' },
            { pattern: /^1\d{10}$/, message: '手机号格式不正确' },
          ]}>
            <Input prefix={<PhoneOutlined style={{ color: '#bfbfbf' }} />} placeholder="手机号" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              发送验证码
            </Button>
          </Form.Item>

          <div style={{ textAlign: 'center', marginTop: 16 }}>
            <Text style={{ fontSize: 13 }}>
              <Link to="/login">返回登录</Link>
            </Text>
          </div>
        </Form>
      )}

      {step === 1 && (
        <Form form={form} onFinish={handleReset} size="large" autoComplete="off">
          <Form.Item name="code" rules={[
            { required: true, message: '请输入验证码' },
            { len: 6, message: '验证码为6位数字' },
          ]}>
            <Input prefix={<SafetyCertificateOutlined style={{ color: '#bfbfbf' }} />} placeholder="验证码（预设：888888）" />
          </Form.Item>

          <Form.Item name="new_password" rules={[
            { required: true, message: '请输入新密码' },
            { min: 8, message: '密码至少 8 个字符' },
          ]}>
            <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="新密码（至少 8 个字符）" />
          </Form.Item>

          <Form.Item
            name="confirm_password"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) return Promise.resolve();
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="确认新密码" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              重置密码
            </Button>
          </Form.Item>

          <div style={{ textAlign: 'center', marginTop: 16 }}>
            <Text style={{ fontSize: 13 }}>
              <Link to="/login">返回登录</Link>
            </Text>
          </div>
        </Form>
      )}
    </div>
  );
}

export default ForgotPassword;
