import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Space, Popconfirm, message, Modal, Form, Input, InputNumber, Switch } from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { listChannels, createChannel, updateChannel, deleteChannel, type ChannelVO, type CreateChannelParams, type UpdateChannelParams } from '@/api/cms';

function ChannelList() {
  const [data, setData] = useState<ChannelVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<ChannelVO | null>(null);
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listChannels();
      setData(res.data.data || []);
    } catch {
      message.error('获取栏目列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleDelete = async (id: number) => {
    try {
      await deleteChannel(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const handleEdit = (channel: ChannelVO) => {
    setEditingChannel(channel);
    form.setFieldsValue({
      name: channel.name,
      slug: channel.slug,
      level: channel.level,
      sort_order: channel.sort_order,
      cover_url: channel.cover_url,
      description: channel.description,
      status: channel.status === 1,
    });
    setModalOpen(true);
  };

  const handleCreate = () => {
    setEditingChannel(null);
    form.resetFields();
    form.setFieldsValue({ level: 0, sort_order: 0, status: true });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const params: CreateChannelParams = {
        name: values.name,
        slug: values.slug || undefined,
        level: values.level || 0,
        sort_order: values.sort_order || 0,
        cover_url: values.cover_url || undefined,
        description: values.description || undefined,
        status: values.status ? 1 : 0,
      };

      if (editingChannel) {
        const updateParams: UpdateChannelParams = { ...params };
        await updateChannel(editingChannel.id, updateParams);
        message.success('更新成功');
      } else {
        await createChannel(params);
        message.success('创建成功');
      }
      setModalOpen(false);
      fetchData();
    } catch (err: any) {
      if (err.errorFields) return; // form validation error
      message.error('操作失败');
    }
  };

  // Flatten tree for table
  const flattenTree = (nodes: ChannelVO[], depth = 0): (ChannelVO & { _depth: number })[] => {
    const result: (ChannelVO & { _depth: number })[] = [];
    for (const node of nodes) {
      result.push({ ...node, _depth: depth, name: '  '.repeat(depth) + node.name } as any);
      if (node.children && node.children.length > 0) {
        result.push(...flattenTree(node.children, depth + 1));
      }
    }
    return result;
  };

  const columns: ColumnsType<ChannelVO & { _depth?: number }> = [
    { title: '栏目名称', dataIndex: 'name', key: 'name', width: 300,
      render: (_text: string, record: ChannelVO & { _depth?: number }) => {
        const prefix = record._depth ? '  '.repeat(record._depth) : '';
        return prefix + (record.name || '');
      },
    },
    { title: 'Slug', dataIndex: 'slug', key: 'slug', width: 150, render: (v: string) => v || '-' },
    { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 80 },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (status: number) => (status === 1 ? <span style={{ color: 'green' }}>显示</span> : <span style={{ color: '#999' }}>隐藏</span>),
    },
    {
      title: '操作', key: 'action', width: 160,
      render: (_: any, record: ChannelVO) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm title="确定删除此栏目?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新建栏目</Button>
      </Space>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={flattenTree(data)}
        loading={loading}
        pagination={false}
      />

      <Modal
        title={editingChannel ? '编辑栏目' : '新建栏目'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="栏目名称" rules={[{ required: true, message: '请输入栏目名称' }]}>
            <Input maxLength={100} />
          </Form.Item>
          <Form.Item name="slug" label="URL 标识">
            <Input maxLength={100} />
          </Form.Item>
          <Form.Item name="level" label="层级">
            <InputNumber min={0} max={10} />
          </Form.Item>
          <Form.Item name="sort_order" label="排序">
            <InputNumber min={0} />
          </Form.Item>
          <Form.Item name="cover_url" label="封面图 URL">
            <Input maxLength={500} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} maxLength={500} />
          </Form.Item>
          <Form.Item name="status" label="显示" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default ChannelList;
