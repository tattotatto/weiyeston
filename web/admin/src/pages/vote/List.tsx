import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Space, Popconfirm, message, Tag, Modal, Progress } from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, BarChartOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { listVotes, deleteVote, getVoteResults, type VoteVO, type VoteOptionVO } from '@/api/vote';

function VoteList() {
  const navigate = useNavigate();
  const [data, setData] = useState<VoteVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [resultModalOpen, setResultModalOpen] = useState(false);
  const [resultData, setResultData] = useState<{ options: VoteOptionVO[]; total: number; title: string } | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listVotes({ page, size: pageSize });
      setData(res.data.data.list || []);
    } catch {
      message.error('获取投票列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleDelete = async (id: number) => {
    try {
      await deleteVote(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const handleViewResults = async (id: number) => {
    try {
      const res = await getVoteResults(id);
      setResultData(res.data.data);
      setResultModalOpen(true);
    } catch {
      message.error('获取结果失败');
    }
  };

  const handleTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current || 1);
    setPageSize(pagination.pageSize || 20);
  };

  const statusMap: Record<number, { color: string; text: string }> = {
    0: { color: 'default', text: '草稿' },
    1: { color: 'green', text: '进行中' },
    2: { color: 'red', text: '已结束' },
  };

  const typeMap: Record<number, string> = {
    1: '单选',
    2: '多选',
  };

  const columns: ColumnsType<VoteVO> = [
    {
      title: '标题', dataIndex: 'title', key: 'title', width: 200,
      render: (title: string) => <strong>{title}</strong>,
    },
    {
      title: '类型', dataIndex: 'vote_type', key: 'vote_type', width: 70,
      render: (v: number) => typeMap[v] || '-',
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (status: number) => {
        const s = statusMap[status] || { color: 'default', text: String(status) };
        return <Tag color={s.color}>{s.text}</Tag>;
      },
    },
    {
      title: '总票数', dataIndex: 'total_votes', key: 'total_votes', width: 80,
    },
    {
      title: '时间', key: 'time', width: 200,
      render: (_: any, record: VoteVO) => (
        <span style={{ fontSize: 12, color: '#999' }}>
          {record.start_time ? dayjs(record.start_time).format('MM-DD HH:mm') : '不限'} ~ {record.end_time ? dayjs(record.end_time).format('MM-DD HH:mm') : '不限'}
        </span>
      ),
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作', key: 'action', width: 220,
      render: (_: any, record: VoteVO) => (
        <Space>
          <Button type="link" size="small" icon={<BarChartOutlined />}
            onClick={() => handleViewResults(record.id)}>结果</Button>
          <Button type="link" size="small" icon={<EditOutlined />}
            onClick={() => navigate(`/votes/${record.id}/edit`)}>编辑</Button>
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
        <Button type="primary" icon={<PlusOutlined />}
          onClick={() => navigate('/votes/create')}>创建投票</Button>
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
        title={resultData ? `投票结果: ${resultData.title}` : '投票结果'}
        open={resultModalOpen}
        onCancel={() => setResultModalOpen(false)}
        footer={null}
        width={500}
      >
        {resultData && (
          <div>
            <p>总投票数: <strong>{resultData.total}</strong></p>
            {resultData.options.map((opt) => {
              const pct = resultData.total > 0 ? Math.round((opt.vote_count / resultData.total) * 100) : 0;
              return (
                <div key={opt.id} style={{ marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <span>{opt.content}</span>
                    <span>{opt.vote_count} 票 ({pct}%)</span>
                  </div>
                  <Progress percent={pct} size="small" />
                </div>
              );
            })}
          </div>
        )}
      </Modal>
    </div>
  );
}

export default VoteList;
