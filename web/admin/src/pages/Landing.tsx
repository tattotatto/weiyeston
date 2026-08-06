import { Button, Layout, Typography, Row, Col, Card, Space } from 'antd';
import {
  WechatOutlined,
  ThunderboltOutlined,
  RobotOutlined,
  SafetyOutlined,
  ApiOutlined,
  AppstoreOutlined,
  LoginOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

const { Header, Content, Footer } = Layout;
const { Title, Paragraph, Text } = Typography;

const features = [
  {
    icon: <WechatOutlined style={{ fontSize: 40, color: '#07c160' }} />,
    title: '一键授权接入',
    desc: '公众号管理员扫码即可完成授权，无需手动填写 AppId/AppSecret。同时兼容手动接入方式。',
  },
  {
    icon: <ThunderboltOutlined style={{ fontSize: 40, color: '#1677ff' }} />,
    title: '可视化页面编辑器',
    desc: '媲美秀米/135编辑器，拖拽式组件排版，实时手机预览，丰富模板库。',
  },
  {
    icon: <RobotOutlined style={{ fontSize: 40, color: '#722ed1' }} />,
    title: 'AI 智能辅助',
    desc: 'AI 帮你写文章、智能排版、自动校对（错别字/语病/敏感词检测）。',
  },
  {
    icon: <AppstoreOutlined style={{ fontSize: 40, color: '#fa8c16' }} />,
    title: '多租户管理',
    desc: '一个账号可管理多个公众号，权限隔离，每个公众号独立配置。',
  },
  {
    icon: <ApiOutlined style={{ fontSize: 40, color: '#13c2c2' }} />,
    title: '完整公众号能力',
    desc: '自定义菜单、关键词自动回复、素材管理、粉丝管理、群发消息全部支持。',
  },
  {
    icon: <SafetyOutlined style={{ fontSize: 40, color: '#52c41a' }} />,
    title: '安全可靠',
    desc: 'Go 高性能后端，JWT 双 Token 认证，Refresh Token Rotation，登录限流。',
  },
];

function Landing() {
  const navigate = useNavigate();

  return (
    <Layout style={{ minHeight: '100vh', background: '#fff' }}>
      <Header
        style={{
          background: '#fff',
          borderBottom: '1px solid #f0f0f0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 80px',
          height: 64,
        }}
      >
        <Space align="center" size={8}>
          <img
            src="https://ynhx.oss-cn-chengdu.aliyuncs.com/%E5%AE%8F%E6%9B%A6%E7%A7%91%E6%8A%80logo-08.png"
            alt="微盈通"
            style={{ width: 36, height: 36, borderRadius: 6 }}
          />
          <Text strong style={{ fontSize: 20, color: '#1677ff' }}>微盈通</Text>
        </Space>
        <Space>
          <Button icon={<LoginOutlined />} onClick={() => navigate('/login')}>
            登录
          </Button>
          <Button type="primary" icon={<UserAddOutlined />} onClick={() => navigate('/register')}>
            免费注册
          </Button>
        </Space>
      </Header>

      <Content>
        {/* Hero */}
        <div
          style={{
            background: 'linear-gradient(135deg, #1677ff 0%, #0958d9 50%, #003eb3 100%)',
            padding: '100px 80px 80px',
            textAlign: 'center',
            color: '#fff',
          }}
        >
          <Title level={1} style={{ color: '#fff', fontSize: 48, marginBottom: 24 }}>
            微信公众号 SaaS 平台
          </Title>
          <Paragraph style={{ color: 'rgba(255,255,255,0.8)', fontSize: 18, maxWidth: 680, margin: '0 auto 40px' }}>
            多租户管理 · 一键授权接入 · 可视化页面编辑 · AI 智能辅助 · Go 高性能引擎
          </Paragraph>
          <Space size="large">
            <Button size="large" ghost onClick={() => navigate('/register')}>
              免费注册
            </Button>
            <Button
              size="large"
              type="primary"
              ghost={false}
              style={{ background: '#fff', color: '#1677ff', border: 'none' }}
              onClick={() => navigate('/login')}
            >
              立即登录
            </Button>
          </Space>
        </div>

        {/* Features */}
        <div style={{ padding: '80px 80px', background: '#f7f8fc' }}>
          <Title level={2} style={{ textAlign: 'center', marginBottom: 12 }}>
            为什么选择微盈通？
          </Title>
          <Paragraph
            type="secondary"
            style={{ textAlign: 'center', maxWidth: 500, margin: '0 auto 48px' }}
          >
            从接入到运营，一站式微信公众号管理解决方案
          </Paragraph>

          <Row gutter={[32, 32]} justify="center" style={{ maxWidth: 1200, margin: '0 auto' }}>
            {features.map((f, i) => (
              <Col key={i} xs={24} sm={12} md={8}>
                <Card
                  hoverable
                  style={{ height: '100%', borderRadius: 12 }}
                  bodyStyle={{ padding: 32, textAlign: 'center' }}
                >
                  <div style={{ marginBottom: 16 }}>{f.icon}</div>
                  <Title level={4}>{f.title}</Title>
                  <Paragraph type="secondary">{f.desc}</Paragraph>
                </Card>
              </Col>
            ))}
          </Row>
        </div>

        {/* CTA */}
        <div style={{ padding: '80px', textAlign: 'center' }}>
          <Title level={2}>开始使用微盈通</Title>
          <Paragraph type="secondary" style={{ marginBottom: 32 }}>
            注册即享试用，无需绑定信用卡
          </Paragraph>
          <Button type="primary" size="large" onClick={() => navigate('/register')}>
            免费注册
          </Button>
        </div>
      </Content>

      <Footer style={{ textAlign: 'center', background: '#f7f8fc' }}>
        <Text type="secondary">云南宏曦科技有限公司版权所有</Text>
      </Footer>
    </Layout>
  );
}

export default Landing;
