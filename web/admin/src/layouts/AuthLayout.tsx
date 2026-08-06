import { Outlet } from 'react-router-dom';
import { Layout, Typography } from 'antd';

const { Content } = Layout;
const { Text } = Typography;

function AuthLayout() {
  return (
    <Layout style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #e8f4fd 0%, #f0f2f5 40%, #e6f0fa 100%)',
      padding: 24,
    }}>
      <Content style={{
        maxWidth: 420,
        width: '100%',
      }}>
        {/* Logo + Brand */}
        <div style={{ textAlign: 'center', marginBottom: 8 }}>
          <div style={{
            width: 64, height: 64, borderRadius: 16,
            background: 'linear-gradient(135deg, #1677ff, #0958d9)',
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 28, color: '#fff', marginBottom: 16,
            boxShadow: '0 4px 12px rgba(22,119,255,0.3)',
          }}>
            🚀
          </div>
          <h1 style={{ fontSize: 24, fontWeight: 700, color: '#1a1a2e', marginBottom: 4 }}>
            微盈通
          </h1>
          <Text type="secondary" style={{ fontSize: 13 }}>
            微信公众号 SaaS 管理平台
          </Text>
        </div>

        {/* Card */}
        <div style={{
          background: '#fff',
          borderRadius: 12,
          padding: '36px 32px',
          boxShadow: '0 2px 16px rgba(0,0,0,0.06)',
        }}>
          <Outlet />
        </div>

        <div style={{ textAlign: 'center', marginTop: 24 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            Go + React + PostgreSQL · 高性能多租户架构
          </Text>
        </div>
      </Content>
    </Layout>
  );
}

export default AuthLayout;
