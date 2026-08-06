import { useState, useEffect } from 'react';
import { Form, Input, Button, Card, Typography, message, Spin } from 'antd';
import { MailOutlined, PhoneOutlined, SmileOutlined, BankOutlined } from '@ant-design/icons';
import { getProfile, updateProfile } from '@/api/auth';
import { useAuthStore } from '@/stores/authStore';

const { Title } = Typography;

interface ProfileFormValues {
  nickname?: string;
  email?: string;
  phone?: string;
  company?: string;
}

function Profile() {
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(true);
  const [form] = Form.useForm();
  const setUser = useAuthStore((s) => s.setUser);

  useEffect(() => {
    loadProfile();
  }, []);

  const loadProfile = async () => {
    setFetching(true);
    try {
      const res = await getProfile();
      const d = res.data;
      form.setFieldsValue({
        nickname: d.nickname || '',
        email: d.email || '',
        phone: d.phone || '',
        company: d.company || '',
      });
    } catch {
      message.error('获取个人资料失败');
    } finally {
      setFetching(false);
    }
  };

  const handleSubmit = async (values: ProfileFormValues) => {
    setLoading(true);
    try {
      const res = await updateProfile({
        nickname: values.nickname || undefined,
        email: values.email || undefined,
        phone: values.phone || undefined,
        company: values.company || undefined,
      });
      message.success('个人资料已更新');
      // Update auth store with new nickname
      const d = res.data;
      setUser({
        id: d.id,
        username: d.username,
        nickname: d.nickname || '',
        role: d.role,
        avatar_url: d.avatar_url,
      });
    } catch (err: unknown) {
      const e = err as { response?: { data?: { msg?: string } } };
      message.error(e.response?.data?.msg || '更新失败');
    } finally {
      setLoading(false);
    }
  };

  if (fetching) {
    return (
      <div style={{ textAlign: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 480, margin: '0 auto' }}>
      <Card>
        <Title level={4} style={{ textAlign: 'center', marginBottom: 24 }}>
          个人资料
        </Title>

        <Form form={form} onFinish={handleSubmit} size="large" layout="vertical">
          <Form.Item
            name="nickname"
            label="昵称"
            rules={[{ max: 50, message: '昵称不能超过 50 个字符' }]}
          >
            <Input prefix={<SmileOutlined style={{ color: '#bfbfbf' }} />} placeholder="昵称" />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ type: 'email', message: '邮箱格式不正确' }]}
          >
            <Input prefix={<MailOutlined style={{ color: '#bfbfbf' }} />} placeholder="邮箱" />
          </Form.Item>

          <Form.Item
            name="phone"
            label="手机号"
            rules={[{ pattern: /^1\d{10}$/, message: '手机号格式不正确' }]}
          >
            <Input prefix={<PhoneOutlined style={{ color: '#bfbfbf' }} />} placeholder="手机号" />
          </Form.Item>

          <Form.Item
            name="company"
            label="公司名称"
            rules={[{ max: 200, message: '公司名称不能超过 200 个字符' }]}
          >
            <Input prefix={<BankOutlined style={{ color: '#bfbfbf' }} />} placeholder="公司名称（可选）" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              保存
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}

export default Profile;
