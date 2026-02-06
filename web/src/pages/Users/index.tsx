import React from 'react';
import { Typography, Card, Empty } from 'antd';

const { Title, Text } = Typography;

const UsersPage: React.FC = () => {
  return (
    <div className="fade-in">
      <div className="page-header">
        <Title level={3} className="page-title">用户管理</Title>
        <Text className="page-description">管理系统用户和权限</Text>
      </div>

      <Card>
        <Empty
          description="用户管理功能开发中"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    </div>
  );
};

export default UsersPage;