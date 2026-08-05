import { Card, Row, Col, Statistic } from 'antd';
import {
  AccountBookOutlined,
  FileTextOutlined,
  BarChartOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';

function Dashboard() {
  return (
    <div>
      <h2>工作台</h2>
      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="公众号"
              value={0}
              prefix={<AccountBookOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="文章"
              value={0}
              prefix={<FileTextOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="投票"
              value={0}
              prefix={<BarChartOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="快讯"
              value={0}
              prefix={<ThunderboltOutlined />}
            />
          </Card>
        </Col>
      </Row>
      <Card style={{ marginTop: 24 }}>
        <p>欢迎使用微盈通 V2 管理后台。功能模块正在开发中，敬请期待。</p>
      </Card>
    </div>
  );
}

export default Dashboard;
