import { useState, useEffect, useCallback } from 'react';
import { Form, Input, Select, Button, Space, message, Card, Row, Col, Switch } from 'antd';
import { SaveOutlined, ArrowLeftOutlined, EyeOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { getArticle, createArticle, updateArticle, type ArticleVO, type CreateArticleParams } from '@/api/cms';
import { listChannels, type ChannelVO } from '@/api/cms';
import { EditorCore } from '@/editor/EditorCore';

const { TextArea } = Input;

function ArticleEdit() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isEdit = !!id;
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [, setArticle] = useState<ArticleVO | null>(null);
  const [content, setContent] = useState<any>({ type: 'doc', content: [] });
  const [channels, setChannels] = useState<ChannelVO[]>([]);

  const fetchChannels = async () => {
    try {
      const res = await listChannels();
      const flat: ChannelVO[] = [];
      const walk = (nodes: ChannelVO[]) => {
        for (const n of nodes) {
          flat.push(n);
          if (n.children) walk(n.children);
        }
      };
      walk(res.data.data || []);
      setChannels(flat);
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    fetchChannels();
    if (isEdit && id) {
      setLoading(true);
      getArticle(Number(id))
        .then((res) => {
          const a = res.data.data;
          setArticle(a);
          form.setFieldsValue({
            title: a.title || '',
            channel_id: a.channel_id,
            author: a.author || '',
            summary: a.summary || '',
            status: a.status === 1,
            is_template: a.is_template,
          });
          if (a.content) {
            setContent(a.content);
          }
        })
        .catch(() => message.error('加载文章失败'))
        .finally(() => setLoading(false));
    }
  }, [id, isEdit, form]);

  const handleSave = async (publishStatus: number) => {
    setSaving(true);
    try {
      const values = await form.validateFields();
      const params: CreateArticleParams = {
        title: values.title || undefined,
        channel_id: values.channel_id || undefined,
        author: values.author || undefined,
        summary: values.summary || undefined,
        content,
        status: publishStatus,
        is_template: values.is_template || false,
      };

      if (isEdit && id) {
        await updateArticle(Number(id), params);
        message.success(publishStatus === 1 ? '已发布' : '草稿已保存');
      } else {
        await createArticle(params);
        message.success(publishStatus === 1 ? '已发布' : '草稿已保存');
        navigate('/cms/articles');
      }
    } catch (err: any) {
      if (err.errorFields) return;
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleContentChange = useCallback((json: any) => {
    setContent(json);
  }, []);

  if (loading) {
    return <div>加载中...</div>;
  }

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/cms/articles')}>返回</Button>
        <Button icon={<SaveOutlined />} loading={saving} onClick={() => handleSave(0)}>保存草稿</Button>
        <Button type="primary" icon={<EyeOutlined />} loading={saving} onClick={() => handleSave(1)}>发布</Button>
      </Space>

      <Row gutter={16}>
        <Col span={18}>
          <Card title="文章内容" size="small">
            <Form form={form} layout="vertical" style={{ marginBottom: 16 }}>
              <Form.Item name="title" label="文章标题">
                <Input placeholder="请输入文章标题" maxLength={200} />
              </Form.Item>
            </Form>
            <EditorCore initialContent={content} onUpdate={handleContentChange} />
          </Card>
        </Col>
        <Col span={6}>
          <Card title="文章设置" size="small">
            <Form form={form} layout="vertical">
              <Form.Item name="channel_id" label="所属栏目">
                <Select
                  allowClear
                  placeholder="选择栏目"
                  options={channels.map(c => ({ label: c.name, value: c.id }))}
                />
              </Form.Item>
              <Form.Item name="author" label="作者">
                <Input placeholder="请输入作者" maxLength={100} />
              </Form.Item>
              <Form.Item name="summary" label="摘要">
                <TextArea rows={3} placeholder="文章摘要/朋友圈分享描述" maxLength={500} />
              </Form.Item>
              <Form.Item name="status" label="发布状态" valuePropName="checked">
                <Switch checkedChildren="已发布" unCheckedChildren="草稿" />
              </Form.Item>
              <Form.Item name="is_template" label="存为模板" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Form>
          </Card>
        </Col>
      </Row>
    </div>
  );
}

export default ArticleEdit;
