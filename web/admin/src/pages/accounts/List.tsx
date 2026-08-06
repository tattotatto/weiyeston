import { useState, useEffect, useCallback } from 'react';
import { Table, Input, Select, Button, Space, Popconfirm, message, Avatar } from 'antd';
import { PlusOutlined, SearchOutlined, DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { listAccounts, deleteAccount, type AccountVO } from '@/api/account';
import AuthTypeTag from './components/AuthTypeTag';
import AuthStatusBadge from './components/AuthStatusBadge';

const { Option } = Select;

function AccountList() {
  const navigate = useNavigate();
  const [data, setData] = useState<AccountVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');
  const [authType, setAuthType] = useState<number | undefined>(undefined);
  const [authStatus, setAuthStatus] = useState<number | undefined>(undefined);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listAccounts({
        page,
        page_size: pageSize,
        keyword: keyword || undefined,
        auth_type: authType,
        auth_status: authStatus !== undefined ? authStatus : -1,
      });
      const { list, total: totalCount } = res.data.data;
      setData(list);
      setTotal(totalCount);
    } catch {
      message.error('获取公众号列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, keyword, authType, authStatus]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleDelete = async (id: number) => {
    try {
      await deleteAccount(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const handleSearch = () => {
    setPage(1);
    fetchData();
  };

  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current || 1);
    setPageSize(pagination.pageSize || 20);
  };

  const maskAppId = (appId: string) => {
    if (appId.length <= 8) return appId;
    return appId.slice(0, 4) + '****' + appId.slice(-4);
  };

  const columns: ColumnsType<AccountVO> = [
    {
      title: '头像',
      dataIndex: 'avatar_url',
      width: 70,
      render: (url: string, record: AccountVO) => (
        <Avatar src={url || record.head_img} shape="square" size={40}>
          {(!url && !record.head_img) ? (record.name?.[0] || '微') : null}
        </Avatar>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 150,
      ellipsis: true,
      render: (name: string, record: AccountVO) => name || record.nick_name || '-',
    },
    {
      title: 'AppId',
      dataIndex: 'wx_app_id',
      width: 140,
      render: (appId: string) => (
        <span title={appId}>{maskAppId(appId)}</span>
      ),
    },
    {
      title: '接入方式',
      dataIndex: 'auth_type',
      width: 100,
      render: (type: number) => <AuthTypeTag authType={type} />,
    },
    {
      title: '状态',
      dataIndex: 'auth_status',
      width: 90,
      render: (status: number) => <AuthStatusBadge authStatus={status} />,
    },
    {
      title: '粉丝数',
      dataIndex: 'fans_count',
      width: 80,
      render: (count: number) => count?.toLocaleString() || '0',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 130,
      render: (date: string) => date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 180,
      render: (_: unknown, record: AccountVO) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/accounts/${record.id}`)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => navigate(`/accounts/${record.id}/edit`)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除该公众号吗？"
            description="删除后数据和配置将被保留但不可使用。"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2>公众号管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/accounts/create')}>
          添加公众号
        </Button>
      </div>

      <div style={{ marginBottom: 16, display: 'flex', gap: 12 }}>
        <Input
          placeholder="搜索公众号名称或 AppId"
          prefix={<SearchOutlined />}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onPressEnter={handleSearch}
          style={{ width: 260 }}
          allowClear
        />
        <Select
          placeholder="接入方式"
          value={authType}
          onChange={(v) => { setAuthType(v); setPage(1); }}
          allowClear
          style={{ width: 130 }}
        >
          <Option value={undefined}>全部</Option>
          <Option value={1}>手动接入</Option>
          <Option value={2}>平台授权</Option>
        </Select>
        <Select
          placeholder="状态"
          value={authStatus}
          onChange={(v) => { setAuthStatus(v); setPage(1); }}
          allowClear
          style={{ width: 130 }}
        >
          <Option value={undefined}>全部</Option>
          <Option value={0}>未接入</Option>
          <Option value={1}>正常</Option>
          <Option value={2}>令牌过期</Option>
          <Option value={3}>已取消</Option>
        </Select>
        <Button type="primary" onClick={handleSearch}>
          搜索
        </Button>
      </div>

      <Table<AccountVO>
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        onChange={handleTableChange}
        onRow={(record) => ({
          style: { cursor: 'pointer' },
          onClick: () => navigate(`/accounts/${record.id}/dashboard`),
        })}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        scroll={{ x: 1000 }}
      />
    </div>
  );
}

export default AccountList;
