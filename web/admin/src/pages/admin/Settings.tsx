import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Tabs,
  message,
  Spin,
} from 'antd';
import {
  getStorageSettings,
  updateStorageSettings,
  type StorageSettingsRequest,
} from '@/api/settings';

const { Option } = Select;

function Settings() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [driver, setDriver] = useState<string>('local');

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getStorageSettings();
      const data = res.data.data;
      setDriver(data.driver || 'local');
      form.setFieldsValue({
        driver: data.driver || 'local',
        local_path: data.local_path || './uploads',
        s3_endpoint: data.s3_endpoint || '',
        s3_bucket: data.s3_bucket || '',
        s3_region: data.s3_region || '',
        s3_key: data.s3_key || '',
        s3_secret: '',
        public_url: data.public_url || '',
      });
    } catch {
      message.error('获取存储配置失败');
    } finally {
      setLoading(false);
    }
  }, [form]);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);

      const req: StorageSettingsRequest = {
        driver: values.driver,
        local_path: values.local_path || '',
        s3_endpoint: values.s3_endpoint || '',
        s3_bucket: values.s3_bucket || '',
        s3_region: values.s3_region || '',
        s3_key: values.s3_key || '',
        s3_secret: values.s3_secret || '',
        public_url: values.public_url || '',
      };

      await updateStorageSettings(req);
      message.success('保存成功');
    } catch (err: unknown) {
      const axiosError = err as { response?: { data?: { msg?: string } } };
      if (axiosError.response?.data?.msg) {
        message.error(axiosError.response.data.msg);
      }
    } finally {
      setSaving(false);
    }
  };

  const handleDriverChange = (value: string) => {
    setDriver(value);
  };

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>系统设置</h2>

      <Spin spinning={loading}>
        <Card title="存储配置">
          <Tabs
            activeKey={driver}
            onChange={handleDriverChange}
            items={[
              {
                key: 'local',
                label: '本地存储',
              },
              {
                key: 's3',
                label: 'S3 兼容存储',
              },
            ]}
          />

          <Form
            form={form}
            layout="vertical"
            style={{ maxWidth: 600, marginTop: 16 }}
          >
            <Form.Item name="driver" hidden>
              <Input />
            </Form.Item>

            <Form.Item
              label="存储驱动"
              name="driver"
              rules={[{ required: true, message: '请选择存储驱动' }]}
            >
              <Select onChange={handleDriverChange}>
                <Option value="local">本地存储</Option>
                <Option value="s3">S3 兼容存储</Option>
              </Select>
            </Form.Item>

            {driver === 'local' && (
              <Form.Item
                label="本地存储路径"
                name="local_path"
                rules={[{ required: true, message: '请输入本地存储路径' }]}
                extra="文件将存储在服务器的此目录下"
              >
                <Input placeholder="./uploads" />
              </Form.Item>
            )}

            {driver === 's3' && (
              <>
                <Form.Item
                  label="S3 Endpoint"
                  name="s3_endpoint"
                  rules={[{ required: true, message: '请输入 S3 Endpoint' }]}
                  extra="如: https://oss-cn-hangzhou.aliyuncs.com"
                >
                  <Input placeholder="https://oss-cn-hangzhou.aliyuncs.com" />
                </Form.Item>

                <Form.Item
                  label="Bucket"
                  name="s3_bucket"
                  rules={[{ required: true, message: '请输入 Bucket 名称' }]}
                >
                  <Input placeholder="my-bucket" />
                </Form.Item>

                <Form.Item
                  label="Region"
                  name="s3_region"
                  rules={[{ required: true, message: '请输入 Region' }]}
                  extra="如: cn-hangzhou, us-east-1"
                >
                  <Input placeholder="cn-hangzhou" />
                </Form.Item>

                <Form.Item
                  label="Access Key"
                  name="s3_key"
                  rules={[{ required: true, message: '请输入 Access Key' }]}
                >
                  <Input placeholder="您的 Access Key ID" />
                </Form.Item>

                <Form.Item
                  label="Secret Key"
                  name="s3_secret"
                  extra="留空则不修改已有密钥。输入新值将覆盖已有密钥"
                >
                  <Input.Password placeholder="您的 Secret Key（留空不修改）" />
                </Form.Item>

                <Form.Item
                  label="自定义 CDN 域名"
                  name="public_url"
                  extra="可选。如: https://cdn.example.com"
                >
                  <Input placeholder="https://cdn.example.com" />
                </Form.Item>
              </>
            )}

            <Form.Item>
              <Button type="primary" onClick={handleSave} loading={saving}>
                保存配置
              </Button>
            </Form.Item>
          </Form>
        </Card>
      </Spin>
    </div>
  );
}

export default Settings;
