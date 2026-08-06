import { useState, useEffect } from 'react';
import { Form, Input, Button, Card, Alert, Upload, message, Space } from 'antd';
import type { FormInstance } from 'antd/es/form';
import { ArrowLeftOutlined, InboxOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { UploadFile } from 'antd/es/upload/interface';
import { createAccount, type CreateAccountParams } from '@/api/account';
import { getServerInfo } from '@/api/server';
import { getToken } from '@/utils/token';

const { Dragger } = Upload;

const normFile = (e: { fileList: UploadFile[] }) => e?.fileList;

function UploadImage({ form, fieldName }: { form: FormInstance; fieldName: string }) {
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const token = getToken();

  return (
    <Dragger
      name="file"
      maxCount={1}
      accept="image/*"
      fileList={fileList}
      action="/api/v1/materials/upload"
      headers={{ Authorization: `Bearer ${token}` }}
      onChange={({ file, fileList: newList }) => {
        setFileList(newList);
        if (file.status === 'done' && file.response?.data?.url) {
          form.setFieldValue(fieldName, file.response.data.url);
          message.success('上传成功');
        } else if (file.status === 'error') {
          message.error('上传失败');
        }
      }}
      onRemove={() => {
        form.setFieldValue(fieldName, '');
        setFileList([]);
      }}
    >
      <p className="ant-upload-drag-icon">
        <InboxOutlined />
      </p>
      <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
      <p className="ant-upload-hint">支持 PNG、JPG、GIF，最大 10MB</p>
    </Dragger>
  );
}

function AccountCreate() {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [serverIp, setServerIp] = useState('');

  useEffect(() => {
    getServerInfo().then(res => {
      const ip = res.data.data?.public_ip;
      if (ip) setServerIp(ip);
    }).catch(() => {});
  }, []);

  const handleSubmit = async (values: CreateAccountParams) => {
    setLoading(true);
    try {
      await createAccount(values);
      message.success('接入成功');
      navigate('/admin/accounts', { replace: true });
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
        <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/admin/accounts')}>
          返回列表
        </Button>
      </div>

      <h2>手动接入公众号</h2>
      <p style={{ color: '#666', marginBottom: 24 }}>
        填写微信公众号的 AppId 和 AppSecret 完成接入
      </p>

      {serverIp && (
        <Alert
          type="warning"
          showIcon
          message="IP 白名单配置"
          description={
            <span>
              请将以下服务器 IP 加入微信公众号后台「开发 → 基本配置 → IP 白名单」：
              <code style={{ fontSize: 16, fontWeight: 'bold', margin: '0 8px', background: '#fff7e6', padding: '2px 8px', borderRadius: 4 }}>
                {serverIp}
              </code>
              否则无法获取 access_token，公众号功能将不可用。
            </span>
          }
          style={{ maxWidth: 640, marginBottom: 24 }}
        />
      )}

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
            label="头像"
            name="avatar_url"
            valuePropName="fileList"
            getValueFromEvent={normFile}
            rules={[{ max: 500, message: 'URL 不能超过 500 个字符' }]}
          >
            <UploadImage fieldName="avatar_url" form={form} />
          </Form.Item>

          <Form.Item
            label="二维码"
            name="qr_code_url"
            valuePropName="fileList"
            getValueFromEvent={normFile}
            rules={[{ max: 500, message: 'URL 不能超过 500 个字符' }]}
          >
            <UploadImage fieldName="qr_code_url" form={form} />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                验证并保存
              </Button>
              <Button onClick={() => navigate('/admin/accounts')}>
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
