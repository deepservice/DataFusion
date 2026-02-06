import React from 'react';
import { Typography, Card, Empty } from 'antd';

const { Title, Text } = Typography;

const ConfigPage: React.FC = () => {
  return (
    <div className="fade-in">
      <div className="page-header">
        <Title level={3} className="page-title">系统配置</Title>
        <Text className="page-description">管理系统配置和参数</Text>
      </div>

      <Card>
        <Empty
          description="系统配置功能开发中"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    </div>
  );
};

export default ConfigPage;