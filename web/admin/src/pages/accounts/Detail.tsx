import { useState, useEffect } from 'react';
import { Descriptions, Button, Space, Spin, Avatar, Card, message } from 'antd';
import { ArrowLeftOutlined, EditOutlined, DeleteOutlined, Popconfirm } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import dayjs from 'dayjs';
import { getAccount, deleteAccount, type AccountVO } from '@/api/account';
import AuthTypeTag from './components/AuthTypeTag';
import AuthStatusBadge from './components/AuthStatusBadge';

function AccountDetail() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [account, setAccount] = useState<AccountVO | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchDetail = async () => {
      if (!id) return;
      setLoading(true);
      try {
        const res = await getAccount(Number(id));
        setAccount(res.data.data);
      } catch {
        message.error('获取公众号详情失败');
      } finally {
        setLoading(false);
      }
    };
    fetchDetail();
  }, [id]);

  const handleDelete = async () => {
    if (!id) return;
    try {
      await deleteAccount(Number(id));
      message.success('删除成功');
      navigate('/accounts', { replace: true });
    } catch {
      message.error('删除失败');
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!account) {
    return <div style={{ textAlign: 'center', padding: 100 }}>公众号不存在</div>;
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/accounts')}>
          返回列表
        </Button>
        <Space>
          <Button
            type="primary"
            icon={<EditOutlined />}
            onClick={() => navigate(`/accounts/${id}/edit`)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除该公众号吗？"
            onConfirm={handleDelete}
            okText="确定"
            cancelText="取消"
          >
            <Button danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      </div>

      <Card style={{ maxWidth: 800 }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 24, gap: 16 }}>
          <Avatar
            src={account.avatar_url || account.head_img}
            shape="square"
            size={64}
          >
            {(!account.avatar_url && !account.head_img) ? (account.name?.[0] || '微') : null}
          </Avatar>
          <div>
            <h2 style={{ margin: 0 }}>{account.name || account.nick_name || '-'}</h2>
            <Space style={{ marginTop: 8 }}>
              <AuthStatusBadge authStatus={account.auth_status} />
              <AuthTypeTag authType={account.auth_type} />
            </Space>
          </div>
        </div>

        <Descriptions title="基本信息" bordered column={2}>
          <Descriptions.Item label="AppId" span={2}>
            <Space>
              <code>{account.wx_app_id}</code>
              <Button
                type="link"
                size="small"
                onClick={() => navigator.clipboard.writeText(account.wx_app_id)}
              >
                Copy
              </Button>
            </Space>
          </Descriptions.Item>
          {account.wx_original_id && (
            <Descriptions.Item label="Original ID">{account.wx_original_id}</Descriptions.Item>
          )}
          {account.description && (
            <Descriptions.Item label="Description" span={2}>{account.description}</Descriptions.Item>
          )}
          <Descriptions.Item label="Fans">
            {account.fans_count?.toLocaleString() || '0'}
          </Descriptions.Item>
          <Descriptions.Item label="Principal">
            {account.principal_name || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Created">
            {account.created_at ? dayjs(account.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Updated">
            {account.updated_at ? dayjs(account.updated_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
          </Descriptions.Item>
        </Descriptions>

        <Descriptions title="Token Status" bordered column={1} style={{ marginTop: 24 }}>
          <Descriptions.Item label="Current Status">
            <AuthStatusBadge authStatus={account.auth_status} />
          </Descriptions.Item>
          {account.token_expire_at && (
            <Descriptions.Item label="Expires At">
              {dayjs(account.token_expire_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
          )}
        </Descriptions>
      </Card>
    </div>
  );
}

export default AccountDetail;
