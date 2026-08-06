import { useState, useEffect, useCallback } from 'react';
import {
  Table, Button, Space, Popconfirm, message, Tag, Modal, Form, Input,
  Select, InputNumber, Switch,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useParams, useNavigate } from 'react-router-dom';
import {
  listReplies, createReply, updateReply, deleteReply,
  type ReplyVO, type CreateReplyParams, type UpdateReplyParams,
} from '@/api/reply';

const { Option } = Select;
const { TextArea } = Input;

function Replies() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const accountId = Number(id);

  const [data, setData] = useState<ReplyVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<ReplyVO | null>(null);
  const [form] = Form.useForm();
  const [replyType, setReplyType] = useState(1);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listReplies(accountId);
      setData(res.data.data || []);
    } catch {
      message.error('获取回复规则列表失败');
    } finally {
      setLoading(false);
    }
  }, [accountId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreate = () => {
    setEditingRule(null);
    setReplyType(1);
    form.resetFields();
    form.setFieldsValue({
      match_type: 0,
      reply_type: 1,
      status: 1,
      sort_order: 0,
    });
    setModalOpen(true);
  };

  const handleEdit = (record: ReplyVO) => {
    setEditingRule(record);
    setReplyType(record.reply_type);
    form.setFieldsValue({
      keyword: record.keyword,
      match_type: record.match_type,
      reply_type: record.reply_type,
      reply_content: record.reply_content,
      reply_title: record.reply_title,
      reply_desc: record.reply_desc,
      reply_cover_url: record.reply_cover_url,
      reply_url: record.reply_url,
      status: record.status,
      sort_order: record.sort_order,
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      if (editingRule) {
        const params: UpdateReplyParams = {};
        if (values.keyword !== undefined) params.keyword = values.keyword || undefined;
        if (values.match_type !== undefined) params.match_type = values.match_type;
        if (values.reply_type !== undefined) params.reply_type = values.reply_type;
        if (values.reply_content !== undefined) params.reply_content = values.reply_content;
        if (values.reply_type === 2) {
          params.reply_title = values.reply_title || undefined;
          params.reply_desc = values.reply_desc || undefined;
          params.reply_cover_url = values.reply_cover_url || undefined;
          params.reply_url = values.reply_url || undefined;
        }
        params.status = values.status ? 1 : 0;
        params.sort_order = values.sort_order;
        await updateReply(editingRule.id, params);
        message.success('规则已更新');
      } else {
        const params: CreateReplyParams = {
          keyword: values.keyword || undefined,
          match_type: values.match_type,
          reply_type: values.reply_type,
          reply_content: values.reply_content,
          status: values.status ? 1 : 0,
          sort_order: values.sort_order || 0,
        };
        if (values.reply_type === 2) {
          params.reply_title = values.reply_title || undefined;
          params.reply_desc = values.reply_desc || undefined;
          params.reply_cover_url = values.reply_cover_url || undefined;
          params.reply_url = values.reply_url || undefined;
        }
        await createReply(accountId, params);
        message.success('规则已创建');
      }

      setModalOpen(false);
      fetchData();
    } catch (err) {
      if (err && typeof err === 'object' && 'errorFields' in err) {
        return; // form validation error
      }
      message.error('操作失败');
    }
  };

  const handleDelete = async (ruleId: number) => {
    try {
      await deleteReply(ruleId);
      message.success('已删除');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const columns: ColumnsType<ReplyVO> = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      width: 120,
      render: (kw: string | null) => kw || <Tag color="blue">默认回复</Tag>,
    },
    {
      title: '匹配方式',
      dataIndex: 'match_type',
      width: 90,
      render: (t: number) => t === 0 ? '精确' : '模糊',
    },
    {
      title: '回复类型',
      dataIndex: 'reply_type',
      width: 90,
      render: (t: number) => <Tag color={t === 1 ? 'green' : 'orange'}>{t === 1 ? '文本' : '图文'}</Tag>,
    },
    {
      title: '回复内容',
      dataIndex: 'reply_content',
      width: 250,
      ellipsis: true,
      render: (content: string, record: ReplyVO) => {
        if (record.reply_type === 1) {
          return content;
        }
        try {
          const items = JSON.parse(content);
          if (Array.isArray(items) && items.length > 0) {
            return `图文回复 (${items.length}条) — ${items[0].title || ''}`;
          }
        } catch {
          return '图文回复';
        }
        return content;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 70,
      render: (s: number) => (
        <Tag color={s === 1 ? 'green' : 'default'}>{s === 1 ? '启用' : '停用'}</Tag>
      ),
    },
    {
      title: '排序',
      dataIndex: 'sort_order',
      width: 70,
    },
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render: (_: unknown, record: ReplyVO) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定删除此规则？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const replyTypeWatch = Form.useWatch('reply_type', form);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/admin/accounts/${accountId}`)}>
            返回
          </Button>
          <h2 style={{ margin: 0 }}>自动回复规则</h2>
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          添加规则
        </Button>
      </div>

      <Table<ReplyVO>
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 800 }}
      />

      <Modal
        title={editingRule ? '编辑回复规则' : '添加回复规则'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        okText="保存"
        cancelText="取消"
        width={640}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ match_type: 0, reply_type: 1, status: true, sort_order: 0 }}>
          <Form.Item name="keyword" label="关键词（留空为默认回复）">
            <Input placeholder="用户发送的关键词" maxLength={200} />
          </Form.Item>

          <Form.Item name="match_type" label="匹配方式">
            <Select>
              <Option value={0}>精确匹配</Option>
              <Option value={1}>包含匹配</Option>
            </Select>
          </Form.Item>

          <Form.Item name="reply_type" label="回复类型">
            <Select onChange={(v) => setReplyType(v)}>
              <Option value={1}>文本回复</Option>
              <Option value={2}>图文回复</Option>
            </Select>
          </Form.Item>

          {replyTypeWatch !== undefined
            ? (replyTypeWatch === 1 || replyType === 1) && (
                <Form.Item
                  name="reply_content"
                  label="回复内容"
                  rules={[{ required: true, message: '请输入回复内容' }]}
                >
                  <TextArea rows={4} placeholder="回复的文本内容" maxLength={2000} showCount />
                </Form.Item>
              )
            : replyType === 1 && (
                <Form.Item
                  name="reply_content"
                  label="回复内容"
                  rules={[{ required: true, message: '请输入回复内容' }]}
                >
                  <TextArea rows={4} placeholder="回复的文本内容" maxLength={2000} showCount />
                </Form.Item>
              )}

          {replyTypeWatch !== undefined
            ? replyTypeWatch === 2 && (
                <>
                  <Form.Item
                    name="reply_content"
                    label="图文内容 JSON"
                    rules={[{ required: true, message: '请输入图文内容' }]}
                    extra='格式: [{"title":"标题","desc":"描述","cover":"封面URL","url":"链接URL"}]'
                  >
                    <TextArea rows={6} placeholder='[{"title":"标题","desc":"描述","cover":"https://...","url":"https://..."}]' />
                  </Form.Item>
                  <Form.Item name="reply_title" label="图文标题（单图文）">
                    <Input placeholder="标题" maxLength={200} />
                  </Form.Item>
                  <Form.Item name="reply_desc" label="图文描述（单图文）">
                    <Input placeholder="描述" maxLength={500} />
                  </Form.Item>
                  <Form.Item name="reply_cover_url" label="封面图片URL（单图文）">
                    <Input placeholder="https://..." maxLength={500} />
                  </Form.Item>
                  <Form.Item name="reply_url" label="原文链接URL（单图文）">
                    <Input placeholder="https://..." maxLength={500} />
                  </Form.Item>
                </>
              )
            : replyType === 2 && (
                <>
                  <Form.Item
                    name="reply_content"
                    label="图文内容 JSON"
                    rules={[{ required: true, message: '请输入图文内容' }]}
                    extra='格式: [{"title":"标题","desc":"描述","cover":"封面URL","url":"链接URL"}]'
                  >
                    <TextArea rows={6} placeholder='[{"title":"标题","desc":"描述","cover":"https://...","url":"https://..."}]' />
                  </Form.Item>
                  <Form.Item name="reply_title" label="图文标题（单图文）">
                    <Input placeholder="标题" maxLength={200} />
                  </Form.Item>
                  <Form.Item name="reply_desc" label="图文描述（单图文）">
                    <Input placeholder="描述" maxLength={500} />
                  </Form.Item>
                  <Form.Item name="reply_cover_url" label="封面图片URL（单图文）">
                    <Input placeholder="https://..." maxLength={500} />
                  </Form.Item>
                  <Form.Item name="reply_url" label="原文链接URL（单图文）">
                    <Input placeholder="https://..." maxLength={500} />
                  </Form.Item>
                </>
              )}

          <Space size="large">
            <Form.Item name="status" label="状态" valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="停用" />
            </Form.Item>
            <Form.Item name="sort_order" label="排序权重">
              <InputNumber min={0} max={9999} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  );
}

export default Replies;
