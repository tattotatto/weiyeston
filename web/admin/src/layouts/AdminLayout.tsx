import { useState, useMemo } from 'react';
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
  UserOutlined,
  MessageOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { removeToken } from '@/utils/token';
import { useAuthStore } from '@/stores/authStore';

const { Header, Sider, Content } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

function AdminLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  const isAdmin = user?.role === 'admin';

  // Extract account ID from path if we are on an account-scoped route
  const pathParts = location.pathname.split('/').filter(Boolean);
  const accountId = pathParts[0] === 'accounts' && pathParts[1] ? pathParts[1] : null;
  const isAccountRoute = accountId !== null && pathParts.length > 2;

  const selectedKey = useMemo(() => {
    // For account-scoped routes, highlight based on the sub-route
    if (isAccountRoute && accountId) {
      const subPath = '/' + pathParts.slice(2).join('/');
      return `/accounts/${accountId}${subPath}`;
    }
    return '/' + location.pathname.split('/').filter(Boolean)[0] || '/dashboard';
  }, [location.pathname, isAccountRoute, accountId, pathParts]);

  const menuItems: MenuItem[] = useMemo(() => {
    if (isAdmin) {
      return [
        {
          key: '/dashboard',
          icon: <DashboardOutlined />,
          label: '工作台',
        },
        {
          key: '/admin/users',
          icon: <UserOutlined />,
          label: '用户管理',
        },
      ];
    }

    // Non-admin: show main menu + account sub-menu when on account route
    const items: MenuItem[] = [
      {
        key: '/dashboard',
        icon: <DashboardOutlined />,
        label: '工作台',
      },
      {
        key: '/accounts',
        icon: <AccountBookOutlined />,
        label: '公众号列表',
      },
    ];

    if (isAccountRoute && accountId) {
      items.push({
        type: 'divider',
      } as MenuItem);
      items.push({
        key: `/accounts/${accountId}/dashboard`,
        icon: <AppstoreOutlined />,
        label: '公众号首页',
      });
      items.push({
        key: `/accounts/${accountId}/cms/channels`,
        icon: <FileTextOutlined />,
        label: '微官网',
      });
      items.push({
        key: `/accounts/${accountId}/news`,
        icon: <ThunderboltOutlined />,
        label: '快讯',
      });
      items.push({
        key: `/accounts/${accountId}/votes`,
        icon: <BarChartOutlined />,
        label: '投票',
      });
      items.push({
        key: `/accounts/${accountId}/replies`,
        icon: <MessageOutlined />,
        label: '自动回复',
      });
      items.push({
        key: `/accounts/${accountId}/menu`,
        icon: <MenuFoldOutlined />,
        label: '自定义菜单',
      });
      items.push({
        key: `/accounts/${accountId}/materials`,
        icon: <PictureOutlined />,
        label: '素材管理',
      });
    }

    return items;
  }, [isAdmin, isAccountRoute, accountId]);

  const onMenuClick: MenuProps['onClick'] = ({ key }) => {
    // For account-scoped sub-menu keys, navigate directly
    navigate(key);
  };

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
          onClick={onMenuClick}
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
            <span>{user?.nickname || user?.username || '管理员'}</span>
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
