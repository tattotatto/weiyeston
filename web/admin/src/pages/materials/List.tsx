import { useState, useEffect, useCallback } from 'react';
import {
  Card, Upload, Select, Button, Popconfirm, message, Pagination, Image, Empty, Spin
} from 'antd';
import {
  DeleteOutlined, PictureOutlined, FileOutlined,
  AudioOutlined, VideoCameraOutlined, InboxOutlined
} from '@ant-design/icons';
import type { RcFile } from 'antd/es/upload/interface';
import { listMaterials, uploadMaterial, deleteMaterial, type MaterialVO } from '@/api/material';
import { listAccounts, type AccountVO } from '@/api/account';

const { Option } = Select;
const { Dragger } = Upload;

const MATERIAL_TYPES = [
  { value: '', label: '全部类型' },
  { value: 'image', label: '图片' },
  { value: 'voice', label: '语音' },
  { value: 'video', label: '视频' },
  { value: 'file', label: '文件' },
];

const TYPE_ICONS: Record<string, React.ReactNode> = {
  image: <PictureOutlined style={{ fontSize: 40, color: '#4a90d9' }} />,
  voice: <AudioOutlined style={{ fontSize: 40, color: '#f5a623' }} />,
  video: <VideoCameraOutlined style={{ fontSize: 40, color: '#e74c3c' }} />,
  file: <FileOutlined style={{ fontSize: 40, color: '#7f8c8d' }} />,
};

function MaterialList() {
  const [accounts, setAccounts] = useState<AccountVO[]>([]);
  const [selectedAccountId, setSelectedAccountId] = useState<number | undefined>(undefined);
  const [materialType, setMaterialType] = useState<string>('');
  const [data, setData] = useState<MaterialVO[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Load accounts for dropdown
  useEffect(() => {
    (async () => {
      try {
        const res = await listAccounts({ page_size: 200 });
        setAccounts(res.data.data.list);
        if (res.data.data.list.length > 0 && !selectedAccountId) {
          setSelectedAccountId(res.data.data.list[0].id);
        }
      } catch {
        // ignore
      }
    })();
  }, []);

  const fetchData = useCallback(async () => {
    if (!selectedAccountId) return;
    setLoading(true);
    try {
      const res = await listMaterials({
        account_id: selectedAccountId,
        type: materialType || undefined,
        page,
        size: pageSize,
      });
      const { list, total: totalCount } = res.data.data;
      setData(list);
      setTotal(totalCount);
    } catch {
      message.error('获取素材列表失败');
    } finally {
      setLoading(false);
    }
  }, [selectedAccountId, materialType, page, pageSize]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleDelete = async (id: number) => {
    try {
      await deleteMaterial(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const handleUpload = async (file: RcFile): Promise<false | void> => {
    if (!selectedAccountId) {
      message.warning('请先选择公众号');
      return false;
    }

    setUploading(true);
    try {
      await uploadMaterial(selectedAccountId, file);
      message.success(`${file.name} 上传成功`);
      setPage(1);
      fetchData();
    } catch {
      message.error(`${file.name} 上传失败`);
    } finally {
      setUploading(false);
    }
    return false; // prevent default upload behavior
  };

  const formatFileSize = (bytes?: number) => {
    if (!bytes) return '';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>素材管理</h2>

      {/* Filters */}
      <div style={{ marginBottom: 16, display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center' }}>
        <Select
          placeholder="选择公众号"
          value={selectedAccountId}
          onChange={(v) => { setSelectedAccountId(v); setPage(1); }}
          style={{ width: 220 }}
          showSearch
          optionFilterProp="label"
          options={accounts.map((a) => ({ value: a.id, label: a.name || a.nick_name || `ID:${a.id}` }))}
        />
        <Select
          placeholder="素材类型"
          value={materialType}
          onChange={(v) => { setMaterialType(v); setPage(1); }}
          style={{ width: 140 }}
        >
          {MATERIAL_TYPES.map((t) => (
            <Option key={t.value} value={t.value}>{t.label}</Option>
          ))}
        </Select>
        <Button type="primary" onClick={fetchData}>刷新</Button>
      </div>

      {/* Upload area */}
      <Card style={{ marginBottom: 16 }}>
        <Dragger
          accept=".jpg,.jpeg,.png,.gif,.bmp,.webp,.svg,.mp3,.wav,.amr,.mp4,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.zip"
          showUploadList={false}
          beforeUpload={handleUpload}
          disabled={uploading || !selectedAccountId}
        >
          <p className="ant-upload-drag-icon">
            <InboxOutlined />
          </p>
          <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
          <p className="ant-upload-hint">
            支持图片、语音、视频、文档等格式
          </p>
        </Dragger>
        {uploading && <Spin style={{ display: 'block', marginTop: 12 }} tip="上传中..." />}
      </Card>

      {/* Material grid */}
      <Spin spinning={loading}>
        {data.length === 0 && !loading ? (
          <Empty description="暂无素材" />
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
              gap: 16,
            }}
          >
            {data.map((item) => (
              <Card
                key={item.id}
                size="small"
                hoverable
                cover={
                  item.type === 'image' ? (
                    <div style={{ height: 160, overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f5f5f5' }}>
                      <Image
                        src={item.url}
                        alt={item.name || ''}
                        style={{ maxHeight: 160, objectFit: 'cover' }}
                        preview={{ mask: '预览' }}
                        fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE2MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSIjZjBmMGYwIi8+PHRleHQgeD0iNTAlIiB5PSI1MCUiIGRvbWluYW50LWJhc2VsaW5lPSJtaWRkbGUiIHRleHQtYW5jaG9yPSJtaWRkbGUiIGZpbGw9IiNjY2MiIGZvbnQtc2l6ZT0iMTQiPuWbvueJh+WKoOi9veWksei0pTwvdGV4dD48L3N2Zz4="
                      />
                    </div>
                  ) : (
                    <div
                      style={{
                        height: 160,
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: '#fafafa',
                      }}
                    >
                      {TYPE_ICONS[item.type] || <FileOutlined style={{ fontSize: 40 }} />}
                      <span style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                        {item.format?.toUpperCase() || item.type}
                      </span>
                    </div>
                  )
                }
                actions={[
                  <Popconfirm
                    key="delete"
                    title="确定要删除该素材吗？"
                    onConfirm={() => handleDelete(item.id)}
                    okText="确定"
                    cancelText="取消"
                  >
                    <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>,
                ]}
              >
                <Card.Meta
                  title={
                    <span style={{ fontSize: 13 }} title={item.name || ''}>
                      {item.name || '未命名'}
                    </span>
                  }
                  description={
                    <div style={{ fontSize: 12, color: '#999' }}>
                      <div>{formatFileSize(item.file_size)}</div>
                      {item.width && item.height && (
                        <div>{item.width}x{item.height}</div>
                      )}
                    </div>
                  }
                />
              </Card>
            ))}
          </div>
        )}
      </Spin>

      {/* Pagination */}
      {total > 0 && (
        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            showSizeChanger
            showTotal={(t) => `共 ${t} 条`}
            onChange={(p, ps) => { setPage(p); setPageSize(ps); }}
          />
        </div>
      )}
    </div>
  );
}

export default MaterialList;
