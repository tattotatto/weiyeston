import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Space, Popconfirm, message, Modal, Form, Input, Switch, Tag } from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import dayjs from 'dayjs';
import { listNews, createNews, updateNews, deleteNews, type NewsVO, type CreateNewsParams, type UpdateNewsParams } from '@/api/news';

const { TextArea } = Input;

function NewsList() {
  const [data, setData] = useState<NewsVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingNews, setEditingNews] = useState<NewsVO | null>(null);
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listNews({ page, size: pageSize });
      setData(res.data.data.list || []);
    } catch {
      message.error('获取快讯列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleDelete = async (id: number) => {
    try {
      await deleteNews(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const handleEdit = (record: NewsVO) => {
    setEditingNews(record);
    form.setFieldsValue({
      channel_id: record.channel_id,
      content: record.content,
      author_name: record.author_name,
      status: record.status === 1,
      is_top: record.is_top,
    });
    setModalOpen(true);
  };

  const handleCreate = () => {
    setEditingNews(null);
    form.resetFields();
    form.setFieldsValue({ channel_id: 1, status: true, is_top: false });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingNews) {
        const params: UpdateNewsParams = {
          channel_id: values.channel_id,
          content: values.content,
          author_name: values.author_name || undefined,
          status: values.status ? 1 : 0,
          is_top: values.is_top || false,
        };
        await updateNews(editingNews.id, params);
        message.success('更新成功');
      } else {
        const params: CreateNewsParams = {
          channel_id: values.channel_id || 1,
          content: values.content,
          author_name: values.author_name || undefined,
          status: values.status ? 1 : 0,
          is_top: values.is_top || false,
        };
        await createNews(params);
        message.success('发布成功');
      }
      setModalOpen(false);
      fetchData();
    } catch (err: any) {
      if (err.errorFields) return;
      message.error('操作失败');
    }
  };

  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current || 1);
    setPageSize(pagination.pageSize || 20);
  };

  const columns: ColumnsType<NewsVO> = [
    {
      title: '内容', dataIndex: 'content', key: 'content', width: 350,
      render: (content: string) => (
        <div style={{ maxWidth: 350, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {content}
        </div>
      ),
    },
    {
      title: '作者', dataIndex: 'author_name', key: 'author_name', width: 100,
      render: (v: string) => v || '管理员',
    },
    {
      title: '置顶', dataIndex: 'is_top', key: 'is_top', width: 70,
      render: (v: boolean) => v ? <Tag color="red">置顶</Tag> : <Tag>否</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (_: number, record: NewsVO) => (
        <Tag color={record.status === 1 ? 'green' : record.status === 2 ? 'orange' : 'default'}>
          {record.status_text}
        </Tag>
      ),
    },
    {
      title: '点赞数', dataIndex: 'like_count', key: 'like_count', width: 80,
    },
    {
      title: '发布时间', dataIndex: 'created_at', key: 'created_at', width: 160,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作', key: 'action', width: 160,
      render: (_: any, record: NewsVO) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>发布快讯</Button>
      </Space>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total: 0,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        onChange={handleTableChange}
      />

      <Modal
        title={editingNews ? '编辑快讯' : '发布快讯'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="content" label="内容" rules={[{ required: true, message: '请输入内容' }]}>
            <TextArea rows={4} maxLength={500} placeholder="请输入快讯内容" showCount />
          </Form.Item>
          <Form.Item name="author_name" label="作者名称">
            <Input placeholder="可选，默认管理员" maxLength={100} />
          </Form.Item>
          <Form.Item name="channel_id" label="栏目 ID">
            <Input type="number" />
          </Form.Item>
          <Form.Item name="status" label="发布状态" valuePropName="checked">
            <Switch checkedChildren="已发布" unCheckedChildren="草稿" />
          </Form.Item>
          <Form.Item name="is_top" label="置顶" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default NewsList;
