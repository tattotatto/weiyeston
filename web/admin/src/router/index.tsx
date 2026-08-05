import { createBrowserRouter, Navigate } from 'react-router-dom';
import AdminLayout from '@/layouts/AdminLayout';
import AuthLayout from '@/layouts/AuthLayout';
import Login from '@/pages/Login';
import Dashboard from '@/pages/Dashboard';
import AccountList from '@/pages/accounts/List';
import AccountCreate from '@/pages/accounts/Create';
import AccountDetail from '@/pages/accounts/Detail';
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

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <AuthLayout />,
    children: [
      { index: true, element: <Login /> },
    ],
  },
  {
    path: '/',
    element: (
      <AuthGuard>
        <AdminLayout />
      </AuthGuard>
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <Dashboard /> },
      { path: 'accounts', element: <AccountList /> },
      { path: 'accounts/create', element: <AccountCreate /> },
      { path: 'accounts/:id', element: <AccountDetail /> },
      { path: 'accounts/:id/replies', element: <Replies /> },
      { path: 'accounts/:id/menu', element: <MenuEditor /> },
      { path: 'materials', element: <MaterialList /> },
      // T12: CMS
      { path: 'cms/channels', element: <ChannelList /> },
      { path: 'cms/articles', element: <ArticleList /> },
      { path: 'cms/articles/create', element: <ArticleEdit /> },
      { path: 'cms/articles/:id/edit', element: <ArticleEdit /> },
      // T13: News
      { path: 'news', element: <NewsList /> },
      // T14: Vote
      { path: 'votes', element: <VoteList /> },
      { path: 'votes/create', element: <VoteCreate /> },
      { path: '*', element: <Navigate to="/dashboard" replace /> },
    ],
  },
]);
