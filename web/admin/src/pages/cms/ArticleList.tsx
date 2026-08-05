import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Space, Popconfirm, message, Select, Tag } from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { listArticles, deleteArticle, type ArticleVO } from '@/api/cms';
import { listChannels, type ChannelVO } from '@/api/cms';

function ArticleList() {
  const navigate = useNavigate();
  const [data, setData] = useState<ArticleVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterChannelId, setFilterChannelId] = useState<number | undefined>(undefined);
  const [filterStatus, setFilterStatus] = useState<number | undefined>(undefined);
  const [channels, setChannels] = useState<ChannelVO[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listArticles({
        page,
        size: pageSize,
        channel_id: filterChannelId,
        status: filterStatus,
      });
      const result = res.data.data;
      setData(result.list || []);
    } catch {
      message.error('获取文章列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filterChannelId, filterStatus]);

  const fetchChannels = useCallback(async () => {
    try {
      const res = await listChannels();
      // Flatten channels for select
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
  }, []);

  useEffect(() => {
    fetchData();
    fetchChannels();
  }, [fetchData, fetchChannels]);

  const handleDelete = async (id: number) => {
    try {
      await deleteArticle(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current || 1);
    setPageSize(pagination.pageSize || 20);
  };

  const handleSearch = () => {
    setPage(1);
    fetchData();
  };

  const statusMap: Record<number, { color: string; text: string }> = {
    0: { color: 'default', text: '草稿' },
    1: { color: 'green', text: '已发布' },
  };

  const columns: ColumnsType<ArticleVO> = [
    {
      title: '标题', dataIndex: 'title', key: 'title', width: 250,
      render: (title: string) => title || '(无标题)',
    },
    {
      title: '栏目', dataIndex: 'channel_id', key: 'channel_id', width: 120,
      render: (cid: number) => {
        const ch = channels.find(c => c.id === cid);
        return ch ? ch.name : '-';
      },
    },
    {
      title: '作者', dataIndex: 'author', key: 'author', width: 100,
      render: (v: string) => v || '-',
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (status: number) => {
        const s = statusMap[status] || { color: 'default', text: String(status) };
        return <Tag color={s.color}>{s.text}</Tag>;
      },
    },
    {
      title: '浏览', dataIndex: 'view_count', key: 'view_count', width: 70,
    },
    {
      title: '发布时间', dataIndex: 'published_at', key: 'published_at', width: 160,
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作', key: 'action', width: 200, fixed: 'right',
      render: (_: any, record: ArticleVO) => (
        <Space>
          <Button type="link" size="small" icon={<EyeOutlined />}
            onClick={() => navigate(`/cms/articles/${record.id}/preview`)}>
            预览
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />}
            onClick={() => navigate(`/cms/articles/${record.id}/edit`)}>
            编辑
          </Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          allowClear
          placeholder="选择栏目"
          style={{ width: 160 }}
          value={filterChannelId}
          onChange={(v) => { setFilterChannelId(v); setPage(1); }}
          options={channels.map(c => ({ label: c.name, value: c.id }))}
        />
        <Select
          allowClear
          placeholder="状态"
          style={{ width: 120 }}
          value={filterStatus}
          onChange={(v) => { setFilterStatus(v); setPage(1); }}
          options={[
            { label: '草稿', value: 0 },
            { label: '已发布', value: 1 },
          ]}
        />
        <Button type="primary" onClick={handleSearch}>查询</Button>
        <Button type="primary" icon={<PlusOutlined />}
          onClick={() => navigate('/cms/articles/create')}>
          新建文章
        </Button>
      </Space>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1100 }}
        pagination={{
          current: page,
          pageSize,
          total: 0,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        onChange={handleTableChange}
      />
    </div>
  );
}

export default ArticleList;
