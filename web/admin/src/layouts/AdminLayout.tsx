import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, theme } from 'antd';
import {
  DashboardOutlined,
  AccountBookOutlined,
  FileTextOutlined,
  ThunderboltOutlined,
  BarChartOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  PictureOutlined,
} from '@ant-design/icons';
import { removeToken } from '@/utils/token';
import { useAuthStore } from '@/stores/authStore';

const { Header, Sider, Content } = Layout;

const menuItems = [
  {
    key: '/dashboard',
    icon: <DashboardOutlined />,
    label: '工作台',
  },
  {
    key: '/accounts',
    icon: <AccountBookOutlined />,
    label: '公众号',
  },
  {
    key: '/cms/channels',
    icon: <FileTextOutlined />,
    label: '微官网',
  },
  {
    key: '/news',
    icon: <ThunderboltOutlined />,
    label: '快讯',
  },
  {
    key: '/votes',
    icon: <BarChartOutlined />,
    label: '投票',
  },
  {
    key: '/materials',
    icon: <PictureOutlined />,
    label: '素材',
  },
];

function AdminLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const logout = useAuthStore((state) => state.logout);
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  const selectedKey = '/' + location.pathname.split('/')[1];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{
          height: 32,
          margin: 16,
          color: '#fff',
          fontSize: collapsed ? 14 : 18,
          fontWeight: 'bold',
          textAlign: 'center',
          overflow: 'hidden',
        }}>
          {collapsed ? '微盈' : '微盈通 V2'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{
          padding: 0,
          background: colorBgContainer,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ fontSize: 16, width: 64, height: 64 }}
          />
          <div style={{ marginRight: 24 }}>
            <span>管理员</span>
            <Button type="link" onClick={() => {
              removeToken();
              logout();
              navigate('/login');
            }}>
              退出
            </Button>
          </div>
        </Header>
        <Content style={{
          margin: 24,
          padding: 24,
          minHeight: 280,
          background: colorBgContainer,
          borderRadius: borderRadiusLG,
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}

export default AdminLayout;
