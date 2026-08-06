import { createBrowserRouter, Navigate } from 'react-router-dom';
import AdminLayout from '@/layouts/AdminLayout';
import AuthLayout from '@/layouts/AuthLayout';
import Landing from '@/pages/Landing';
import Login from '@/pages/Login';
import Register from '@/pages/Register';
import Dashboard from '@/pages/Dashboard';
import AccountList from '@/pages/accounts/List';
import AccountCreate from '@/pages/accounts/Create';
import AccountDetail from '@/pages/accounts/Detail';
import AccountDashboard from '@/pages/accounts/Dashboard';
import Replies from '@/pages/accounts/Replies';
import MenuEditor from '@/pages/accounts/MenuEditor';
import MaterialList from '@/pages/materials/List';
import AuthGuard from '@/components/common/AuthGuard';
// T12: CMS
import ChannelList from '@/pages/cms/ChannelList';
import ArticleList from '@/pages/cms/ArticleList';
import ArticleEdit from '@/pages/cms/ArticleEdit';
// T13: News
import NewsList from '@/pages/news/List';
// T14: Vote
import VoteList from '@/pages/vote/List';
import VoteCreate from '@/pages/vote/Create';
// Admin
import UserList from '@/pages/admin/UserList';
import Settings from '@/pages/admin/Settings';

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <AuthLayout />,
    children: [
      { index: true, element: <Login /> },
    ],
  },
  {
    path: '/register',
    element: <AuthLayout />,
    children: [
      { index: true, element: <Register /> },
    ],
  },
  // Landing page (public)
  { path: '/', element: <Landing /> },
  // Admin (protected)
  {
    path: '/admin',
    element: (
      <AuthGuard>
        <AdminLayout />
      </AuthGuard>
    ),
    children: [
      { index: true, element: <Navigate to="/admin/dashboard" replace /> },
      { path: 'dashboard', element: <Dashboard /> },
      { path: 'accounts', element: <AccountList /> },
      { path: 'accounts/create', element: <AccountCreate /> },
      { path: 'accounts/:id', element: <AccountDetail /> },
      { path: 'accounts/:id/dashboard', element: <AccountDashboard /> },
      { path: 'accounts/:id/replies', element: <Replies /> },
      { path: 'accounts/:id/menu', element: <MenuEditor /> },
      // Account-scoped sub-routes (new)
      { path: 'accounts/:id/cms/channels', element: <ChannelList /> },
      { path: 'accounts/:id/cms/articles', element: <ArticleList /> },
      { path: 'accounts/:id/cms/articles/create', element: <ArticleEdit /> },
      { path: 'accounts/:id/cms/articles/:articleId/edit', element: <ArticleEdit /> },
      { path: 'accounts/:id/news', element: <NewsList /> },
      { path: 'accounts/:id/votes', element: <VoteList /> },
      { path: 'accounts/:id/votes/create', element: <VoteCreate /> },
      { path: 'accounts/:id/materials', element: <MaterialList /> },
      // Global routes (backward compatible)
      { path: 'materials', element: <MaterialList /> },
      { path: 'cms/channels', element: <ChannelList /> },
      { path: 'cms/articles', element: <ArticleList /> },
      { path: 'cms/articles/create', element: <ArticleEdit /> },
      { path: 'cms/articles/:id/edit', element: <ArticleEdit /> },
      { path: 'news', element: <NewsList /> },
      { path: 'votes', element: <VoteList /> },
      { path: 'votes/create', element: <VoteCreate /> },
      // Super admin
      { path: 'users', element: <UserList /> },
      { path: 'settings', element: <Settings /> },
      { path: '*', element: <Navigate to="/admin/dashboard" replace /> },
    ],
  },
]);
