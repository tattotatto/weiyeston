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
          <img
            src="https://ynhx.oss-cn-chengdu.aliyuncs.com/%E5%AE%8F%E6%9B%A6%E7%A7%91%E6%8A%80logo-08.png"
            alt="微盈通"
            style={{ width: 72, height: 72, marginBottom: 16, borderRadius: 12 }}
          />
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
            云南宏曦科技有限公司版权所有
          </Text>
        </div>
      </Content>
    </Layout>
  );
}

export default AuthLayout;
