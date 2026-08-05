import { Tag } from 'antd';
import { SafetyCertificateOutlined, LinkOutlined } from '@ant-design/icons';

interface AuthTypeTagProps {
  authType: number;
}

function AuthTypeTag({ authType }: AuthTypeTagProps) {
  if (authType === 1) {
    return (
      <Tag icon={<SafetyCertificateOutlined />} color="blue">
        手动接入
      </Tag>
    );
  }
  if (authType === 2) {
    return (
      <Tag icon={<LinkOutlined />} color="green">
        平台授权
      </Tag>
    );
  }
  return <Tag color="default">未知</Tag>;
}

export default AuthTypeTag;
