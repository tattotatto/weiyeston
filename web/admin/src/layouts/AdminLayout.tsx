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
  SettingOutlined,
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
    const path = location.pathname;
    // Return the full path as key for menu matching
    return path;
  }, [location.pathname]);

  const menuItems = (): MenuItem[] => {
    if (isAdmin) {
      return [
        { key: '/admin/dashboard', icon: <DashboardOutlined />, label: '工作台' },
        { key: '/admin/users', icon: <UserOutlined />, label: '用户管理' },
        { key: '/admin/settings', icon: <SettingOutlined />, label: '系统设置' },
      ];
    }

    const items: MenuItem[] = [
      { key: '/admin/dashboard', icon: <DashboardOutlined />, label: '工作台' },
      { key: '/admin/accounts', icon: <AccountBookOutlined />, label: '公众号列表' },
    ];

    if (isAccountRoute && accountId) {
      const prefix = `/admin/accounts/${accountId}`;
      items.push(
        { key: `${prefix}/dashboard`, icon: <AppstoreOutlined />, label: '公众号首页' },
        { key: `${prefix}/cms/channels`, icon: <FileTextOutlined />, label: '微官网' },
        { key: `${prefix}/news`, icon: <ThunderboltOutlined />, label: '快讯' },
        { key: `${prefix}/votes`, icon: <BarChartOutlined />, label: '投票' },
        { key: `${prefix}/replies`, icon: <MessageOutlined />, label: '自动回复' },
        { key: `${prefix}/menu`, icon: <MenuFoldOutlined />, label: '自定义菜单' },
        { key: `${prefix}/materials`, icon: <PictureOutlined />, label: '素材管理' },
      );
    }
    return items;
  };

  const onMenuClick: MenuProps['onClick'] = ({ key }) => {
    // For account-scoped sub-menu keys, navigate directly
    navigate(key);
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 10,
          height: 48,
          margin: '16px 16px 8px',
          overflow: 'hidden',
        }}>
          <img
            src="https://ynhx.oss-cn-chengdu.aliyuncs.com/%E5%AE%8F%E6%9B%A6%E7%A7%91%E6%8A%80logo-08.png"
            alt=""
            style={{ width: 32, height: 32, borderRadius: 6, flexShrink: 0 }}
          />
          {!collapsed && (
            <span style={{ color: '#fff', fontSize: 16, fontWeight: 'bold', whiteSpace: 'nowrap' }}>
              微盈通 V2
            </span>
          )}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems()}
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
          <div style={{ marginRight: 24, display: 'flex', alignItems: 'center', gap: 8 }}>
            <span>{user?.nickname || user?.username || '管理员'}</span>
            <Button type="link" onClick={() => navigate('/admin/change-password')}>
              修改密码
            </Button>
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
