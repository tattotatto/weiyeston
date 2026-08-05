import { Badge } from 'antd';

interface AuthStatusBadgeProps {
  authStatus: number;
}

const statusMap: Record<number, { status: 'success' | 'warning' | 'error' | 'default'; text: string }> = {
  0: { status: 'default', text: '未接入' },
  1: { status: 'success', text: '正常' },
  2: { status: 'warning', text: '令牌过期' },
  3: { status: 'error', text: '已取消' },
};

function AuthStatusBadge({ authStatus }: AuthStatusBadgeProps) {
  const config = statusMap[authStatus] || { status: 'default' as const, text: '未知' };
  return <Badge status={config.status} text={config.text} />;
}

export default AuthStatusBadge;
