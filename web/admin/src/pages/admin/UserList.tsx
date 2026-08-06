import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Modal, Form, Select, DatePicker, Tag, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { getUsers, updateUser, type AdminUserVO } from '@/api/admin';

const { Option } = Select;

const statusMap: Record<number, { color: string; label: string }> = {
  0: { color: 'orange', label: '待审核' },
  1: { color: 'green', label: '已通过' },
  2: { color: 'red', label: '已禁用' },
};

const vipLevelOptions = [
  { value: 'trial', label: '试用' },
  { value: 'basic', label: '基础' },
  { value: 'pro', label: '专业' },
  { value: 'enterprise', label: '企业' },
];

function UserList() {
  const [data, setData] = useState<AdminUserVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<AdminUserVO | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getUsers();
      const result = res.data.data as unknown as { list: AdminUserVO[]; total: number };
      setData(result.list || []);
    } catch {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleEdit = (record: AdminUserVO) => {
    setEditingUser(record);
    form.setFieldsValue({
      status: record.status,
      vip_level: record.vip_level,
      vip_end_time: record.vip_end_time ? dayjs(record.vip_end_time) : null,
    });
    setEditModalOpen(true);
  };

  const handleSave = async () => {
    if (!editingUser) return;
    try {
      const values = await form.validateFields();
      setSaving(true);
      await updateUser(editingUser.id, {
        status: values.status,
        vip_level: values.vip_level,
        vip_end_time: values.vip_end_time
          ? values.vip_end_time.format('YYYY-MM-DD HH:mm:ss')
          : undefined,
      });
      message.success('更新成功');
      setEditModalOpen(false);
      fetchData();
    } catch (err: unknown) {
      const axiosError = err as { response?: { data?: { msg?: string } } };
      if (axiosError.response?.data?.msg) {
        message.error(axiosError.response.data.msg);
      }
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<AdminUserVO> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: 120,
    },
    {
      title: '昵称',
      dataIndex: 'nickname',
      width: 120,
      render: (nickname: string) => nickname || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: number) => {
        const info = statusMap[status] || { color: 'default', label: '未知' };
        return <Tag color={info.color}>{info.label}</Tag>;
      },
    },
    {
      title: 'VIP 等级',
      dataIndex: 'vip_level',
      width: 100,
      render: (level: string) => {
        const opt = vipLevelOptions.find((o) => o.value === level);
        return opt ? opt.label : level || '-';
      },
    },
    {
      title: 'VIP 到期',
      dataIndex: 'vip_end_time',
      width: 140,
      render: (date: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 140,
      render: (date: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      render: (_: unknown, record: AdminUserVO) => (
        <Button type="link" size="small" onClick={() => handleEdit(record)}>
          编辑
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2>用户管理</h2>
        <Button icon={<ReloadOutlined />} onClick={fetchData}>
          刷新
        </Button>
      </div>

      <Table<AdminUserVO>
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 900 }}
      />

      <Modal
        title="编辑用户"
        open={editModalOpen}
        onOk={handleSave}
        onCancel={() => setEditModalOpen(false)}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            label="状态"
            name="status"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select placeholder="请选择状态">
              <Option value={0}>待审核</Option>
              <Option value={1}>已通过</Option>
              <Option value={2}>已禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item
            label="VIP 等级"
            name="vip_level"
            rules={[{ required: true, message: '请选择 VIP 等级' }]}
          >
            <Select placeholder="请选择 VIP 等级">
              {vipLevelOptions.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label="VIP 到期时间" name="vip_end_time">
            <DatePicker showTime style={{ width: '100%' }} placeholder="选择到期时间" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default UserList;
