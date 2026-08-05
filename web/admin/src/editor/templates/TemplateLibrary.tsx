import { useState, useMemo } from 'react'
import { Card, Tabs, Tag, Button, Modal, Empty, message, Spin } from 'antd'
import {
  PictureOutlined,
  CheckOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { PRESET_TEMPLATES, PresetTemplate } from './templates'

interface TemplateLibraryProps {
  onApplyTemplate: (template: PresetTemplate) => void
  visible: boolean
  onClose: () => void
}

export function TemplateLibrary({ onApplyTemplate, visible, onClose }: TemplateLibraryProps) {
  const [activeTab, setActiveTab] = useState<string>('全部')
  const [previewTemplate, setPreviewTemplate] = useState<PresetTemplate | null>(null)
  const [loading, setLoading] = useState(false)

  const categories: { key: string; label: string }[] = [
    { key: '全部', label: '全部' },
    { key: '节日', label: '节日' },
    { key: '活动', label: '活动' },
    { key: '新闻', label: '新闻' },
  ]

  const filteredTemplates = useMemo(() => {
    if (activeTab === '全部') {
      return PRESET_TEMPLATES
    }
    return PRESET_TEMPLATES.filter((t) => t.category === activeTab)
  }, [activeTab])

  const handleApply = async (template: PresetTemplate) => {
    setLoading(true)
    try {
      onApplyTemplate(template)
      message.success(`已应用模板：${template.name}`)
      onClose()
    } catch {
      message.error('应用模板失败')
    } finally {
      setLoading(false)
    }
  }

  const handlePreview = (template: PresetTemplate) => {
    setPreviewTemplate(template)
  }

  const handleClosePreview = () => {
    setPreviewTemplate(null)
  }

  const categoryColorMap: Record<string, string> = {
    '节日': 'red',
    '活动': 'orange',
    '新闻': 'blue',
  }

  const renderTemplateGrid = () => {
    if (filteredTemplates.length === 0) {
      return <Empty description={`暂无"${activeTab}"类别的模板`} />
    }

    return (
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
        {filteredTemplates.map((tmpl) => (
          <Card
            key={tmpl.id}
            hoverable
            cover={
              tmpl.coverUrl ? (
                <img
                  src={tmpl.coverUrl}
                  alt={tmpl.name}
                  style={{ height: 120, objectFit: 'cover' }}
                />
              ) : (
                <div
                  style={{
                    height: 120,
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 48,
                    color: '#fff',
                    fontWeight: 'bold',
                  }}
                >
                  {tmpl.name.slice(0, 1)}
                </div>
              )
            }
            actions={[
              <SearchOutlined key="preview" onClick={() => handlePreview(tmpl)} />,
              <CheckOutlined key="apply" onClick={() => handleApply(tmpl)} />,
            ]}
            style={{ borderRadius: 8 }}
          >
            <Card.Meta
              title={tmpl.name}
              description={
                <div>
                  <Tag color={categoryColorMap[tmpl.category] || 'default'} style={{ marginBottom: 4 }}>
                    {tmpl.category}
                  </Tag>
                  <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>{tmpl.description}</div>
                  <div style={{ marginTop: 6 }}>
                    {tmpl.tags.map((tag) => (
                      <Tag key={tag} style={{ fontSize: 11, marginBottom: 4 }}>
                        {tag}
                      </Tag>
                    ))}
                  </div>
                </div>
              }
            />
          </Card>
        ))}
      </div>
    )
  }

  return (
    <>
      <Modal
        title="模板库"
        open={visible}
        onCancel={onClose}
        width={840}
        footer={null}
        destroyOnClose
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={categories.map((cat) => ({
            key: cat.key,
            label: cat.label,
            children: (
              <div style={{ minHeight: 300 }}>
                <Spin spinning={loading}>{renderTemplateGrid()}</Spin>
              </div>
            ),
          }))}
        />
      </Modal>

      <Modal
        title={`模板预览 - ${previewTemplate?.name ?? ''}`}
        open={previewTemplate !== null}
        onCancel={handleClosePreview}
        width={700}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={handleClosePreview}>关闭</Button>
            <Button
              type="primary"
              icon={<CheckOutlined />}
              loading={loading}
              onClick={() => previewTemplate && handleApply(previewTemplate)}
            >
              应用模板
            </Button>
          </div>
        }
      >
        {previewTemplate && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Tag color={categoryColorMap[previewTemplate.category] || 'default'}>
                {previewTemplate.category}
              </Tag>
              {previewTemplate.tags.map((tag) => (
                <Tag key={tag}>{tag}</Tag>
              ))}
            </div>
            <p style={{ color: '#666', marginBottom: 16 }}>{previewTemplate.description}</p>
            <div
              style={{
                border: '1px solid #f0f0f0',
                borderRadius: 8,
                padding: 16,
                background: '#fafafa',
                maxHeight: 400,
                overflow: 'auto',
              }}
            >
              <div style={{ fontWeight: 'bold', marginBottom: 12, fontSize: 14, color: '#999' }}>
                包含 {previewTemplate.content.content ? (previewTemplate.content.content as unknown[]).length : 0} 个内容模块
              </div>
              {(previewTemplate.content.content as unknown[]) &&
                (previewTemplate.content.content as Array<{ type: string; attrs?: Record<string, unknown> }>).map(
                  (block, idx) => (
                    <div
                      key={idx}
                      style={{
                        padding: '8px 12px',
                        marginBottom: 8,
                        background: '#fff',
                        borderRadius: 4,
                        border: '1px solid #f0f0f0',
                        fontSize: 13,
                      }}
                    >
                      <Tag style={{ marginRight: 8 }}>{block.type}</Tag>
                      {block.attrs?.text ? (
                        <span style={{ color: '#666' }}>
                          {(block.attrs.text as string).slice(0, 40)}
                        </span>
                      ) : block.attrs?.src !== undefined ? (
                        <span style={{ color: '#999' }}>
                          <PictureOutlined /> 图片
                        </span>
                      ) : (
                        <span style={{ color: '#999' }}>{block.type} 模块</span>
                      )}
                    </div>
                  )
                )}
            </div>
          </div>
        )}
      </Modal>
    </>
  )
}
