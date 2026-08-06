import { useState, useEffect } from 'react';
import { Card, Row, Col, Spin, message, Typography, Descriptions } from 'antd';
import {
  FileTextOutlined,
  ThunderboltOutlined,
  BarChartOutlined,
  MessageOutlined,
  MenuOutlined,
  PictureOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import dayjs from 'dayjs';
import { getAccount, type AccountVO } from '@/api/account';

const { Title, Text } = Typography;

const subModules = [
  {
    key: 'cms',
    title: '微官网',
    description: '管理微官网频道与文章',
    icon: <FileTextOutlined style={{ fontSize: 32, color: '#1677ff' }} />,
    path: 'cms/channels',
  },
  {
    key: 'news',
    title: '快讯',
    description: '管理快讯内容',
    icon: <ThunderboltOutlined style={{ fontSize: 32, color: '#fa8c16' }} />,
    path: 'news',
  },
  {
    key: 'votes',
    title: '投票',
    description: '创建和管理投票活动',
    icon: <BarChartOutlined style={{ fontSize: 32, color: '#52c41a' }} />,
    path: 'votes',
  },
  {
    key: 'replies',
    title: '自动回复',
    description: '配置关键词自动回复',
    icon: <MessageOutlined style={{ fontSize: 32, color: '#722ed1' }} />,
    path: 'replies',
  },
  {
    key: 'menu',
    title: '自定义菜单',
    description: '管理公众号底部菜单',
    icon: <MenuOutlined style={{ fontSize: 32, color: '#eb2f96' }} />,
    path: 'menu',
  },
  {
    key: 'materials',
    title: '素材管理',
    description: '管理图文和图片素材',
    icon: <PictureOutlined style={{ fontSize: 32, color: '#13c2c2' }} />,
    path: 'materials',
  },
];

function AccountDashboard() {
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
        message.error('获取公众号信息失败');
      } finally {
        setLoading(false);
      }
    };
    fetchDetail();
  }, [id]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!account) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <p>公众号不存在</p>
        <a onClick={() => navigate('/admin/accounts')}>返回列表</a>
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', alignItems: 'center', gap: 16 }}>
        <ArrowLeftOutlined
          style={{ cursor: 'pointer', fontSize: 16 }}
          onClick={() => navigate('/admin/accounts')}
        />
        <div>
          <Title level={3} style={{ margin: 0 }}>
            {account.name || account.nick_name || '未命名公众号'}
          </Title>
          <Text type="secondary">公众号管理首页</Text>
        </div>
      </div>

      <Card style={{ marginBottom: 24, maxWidth: 800 }}>
        <Descriptions size="small" column={2}>
          <Descriptions.Item label="AppId">
            <code>{account.wx_app_id}</code>
          </Descriptions.Item>
          <Descriptions.Item label="粉丝数">
            {account.fans_count?.toLocaleString() || '0'}
          </Descriptions.Item>
          <Descriptions.Item label="接入方式">
            {account.auth_type === 1 ? '手动接入' : '平台授权'}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {account.created_at ? dayjs(account.created_at).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Title level={4} style={{ marginTop: 24, marginBottom: 16 }}>
        功能模块
      </Title>

      <Row gutter={[16, 16]}>
        {subModules.map((mod) => (
          <Col key={mod.key} xs={24} sm={12} lg={8}>
            <Card
              hoverable
              onClick={() => navigate(`/admin/accounts/${id}/${mod.path}`)}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                {mod.icon}
                <div>
                  <div style={{ fontSize: 16, fontWeight: 500, marginBottom: 4 }}>
                    {mod.title}
                  </div>
                  <Text type="secondary" style={{ fontSize: 13 }}>
                    {mod.description}
                  </Text>
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}

export default AccountDashboard;
