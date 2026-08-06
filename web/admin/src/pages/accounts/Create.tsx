import { useState } from 'react';
import { Form, Input, Button, Card, message, Space } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { createAccount, type CreateAccountParams } from '@/api/account';
import { getServerInfo } from '@/api/server';

function AccountCreate() {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: CreateAccountParams) => {
    setLoading(true);
    try {
      await createAccount(values);
      message.success('接入成功');

      // 获取服务器公网 IP 并提示配置白名单
      try {
        const serverRes = await getServerInfo();
        const publicIp = serverRes.data.data?.public_ip;
        if (publicIp) {
          message.info(
            `请将以下IP加入微信公众号IP白名单: ${publicIp}`,
            8,
          );
        }
      } catch {
        // IP 获取失败不影响主流程
      }

      navigate('/accounts', { replace: true });
    } catch (err: unknown) {
      const axiosError = err as { response?: { data?: { msg?: string } } };
      const msg = axiosError.response?.data?.msg || '接入失败，请重试';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/accounts')}>
          返回列表
        </Button>
      </div>

      <h2>手动接入公众号</h2>
      <p style={{ color: '#666', marginBottom: 24 }}>
        填写微信公众号的 AppId 和 AppSecret 完成接入
      </p>

      <Card style={{ maxWidth: 640 }}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          requiredMark="optional"
        >
          <Form.Item
            label="公众号名称"
            name="name"
            rules={[
              { required: true, message: '请输入公众号名称' },
              { max: 100, message: '名称不能超过 100 个字符' },
            ]}
          >
            <Input placeholder="例如：云南农担" />
          </Form.Item>

          <Form.Item
            label="AppId"
            name="wx_app_id"
            rules={[
              { required: true, message: '请输入 AppId' },
              { pattern: /^wx[a-zA-Z0-9]{16,18}$/, message: 'AppId 格式不正确，应以 wx 开头' },
            ]}
            extra="以 wx 开头的开发者 ID，在微信公众号后台「开发 > 基本配置」获取"
          >
            <Input placeholder="wx_________________" />
          </Form.Item>

          <Form.Item
            label="AppSecret"
            name="wx_app_secret"
            rules={[
              { required: true, message: '请输入 AppSecret' },
              { max: 200, message: 'AppSecret 不能超过 200 个字符' },
            ]}
            extra="开发者密钥，请妥善保管"
          >
            <Input.Password placeholder="请输入 AppSecret" visibilityToggle />
          </Form.Item>

          <Form.Item
            label="原始 ID"
            name="wx_original_id"
            rules={[{ max: 50, message: '不能超过 50 个字符' }]}
            extra="以 gh_ 开头的微信号原始 ID（可选）"
          >
            <Input placeholder="gh_xxxx" />
          </Form.Item>

          <Form.Item
            label="公众号简介"
            name="description"
            rules={[{ max: 500, message: '简介不能超过 500 个字符' }]}
          >
            <Input.TextArea rows={3} placeholder="公众号简介（可选）" />
          </Form.Item>

          <Form.Item
            label="头像 URL"
            name="avatar_url"
            rules={[{ max: 500, message: 'URL 不能超过 500 个字符' }]}
          >
            <Input placeholder="https://" />
          </Form.Item>

          <Form.Item
            label="二维码 URL"
            name="qr_code_url"
            rules={[{ max: 500, message: 'URL 不能超过 500 个字符' }]}
          >
            <Input placeholder="https://" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                验证并保存
              </Button>
              <Button onClick={() => navigate('/accounts')}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}

export default AccountCreate;
