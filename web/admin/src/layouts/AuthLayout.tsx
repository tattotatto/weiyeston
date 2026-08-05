import { Outlet } from 'react-router-dom';
import { Layout } from 'antd';

const { Content } = Layout;

function AuthLayout() {
  return (
    <Layout style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#f0f2f5',
    }}>
      <Content style={{
        maxWidth: 400,
        width: '100%',
        padding: 24,
      }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <h1 style={{ fontSize: 28, color: '#1677ff' }}>微盈通 V2</h1>
          <p style={{ color: '#888' }}>微信公众号管理平台</p>
        </div>
        <Outlet />
      </Content>
    </Layout>
  );
}

export default AuthLayout;
