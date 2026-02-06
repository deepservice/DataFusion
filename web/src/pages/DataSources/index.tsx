import React from 'react';
import { Typography, Card, Empty } from 'antd';

const { Title, Text } = Typography;

const DataSourcesPage: React.FC = () => {
  return (
    <div className="fade-in">
      <div className="page-header">
        <Title level={3} className="page-title">数据源管理</Title>
        <Text className="page-description">管理数据采集的数据源配置</Text>
      </div>

      <Card>
        <Empty
          description="数据源管理功能开发中"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    </div>
  );
};

export default DataSourcesPage;