import { useState } from 'react'
import {
  Tabs,
  Form,
  Input,
  Button,
  Select,
  InputNumber,
  Spin,
  Alert,
  List,
  Tag,
  Typography,
  message,
} from 'antd'
import {
  RobotOutlined,
  FormatPainterOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import { aiWrite, aiLayout, aiProofread, Correction } from '@/api/ai'
import type { Editor } from '@tiptap/core'

const { TextArea } = Input
const { Text } = Typography

interface AIPanelProps {
  editor: Editor | null
  visible: boolean
  onClose: () => void
}

interface WriteFormValues {
  title: string
  keywords?: string
  style?: string
  max_words?: number
}

export function AIPanel({ editor, visible, onClose }: AIPanelProps) {
  const [loading, setLoading] = useState(false)
  const [writeResult, setWriteResult] = useState<string>('')
  const [layoutResult, setLayoutResult] = useState<Record<string, unknown> | null>(null)
  const [corrections, setCorrections] = useState<Correction[]>([])
  const [proofreadDone, setProofreadDone] = useState(false)

  const handleWrite = async (values: WriteFormValues) => {
    setLoading(true)
    try {
      const result = await aiWrite({
        title: values.title,
        keywords: values.keywords,
        style: values.style || '正式',
        max_words: values.max_words || 500,
      })
      setWriteResult(result.content)
      message.success('文章生成成功')
    } catch {
      // aiWrite already shows error toast via api/ai.ts
    } finally {
      setLoading(false)
    }
  }

  const handleLayout = async () => {
    if (!editor) {
      message.warning('编辑器未就绪')
      return
    }
    setLoading(true)
    try {
      const json = JSON.stringify(editor.getJSON())
      const result = await aiLayout({ content: json })
      setLayoutResult(result.content)
      editor.commands.setContent(result.content)
      message.success('智能排版完成')
    } catch {
      // aiLayout already shows error toast via api/ai.ts
    } finally {
      setLoading(false)
    }
  }

  const handleProofread = async () => {
    if (!editor) {
      message.warning('编辑器未就绪')
      return
    }
    const text = editor.getText()
    if (!text || text.trim().length === 0) {
      message.warning('请先输入文本内容')
      return
    }
    setLoading(true)
    try {
      const result = await aiProofread({ text })
      setCorrections(result.corrections)
      setProofreadDone(true)
      if (result.corrections.length === 0) {
        message.success('未发现问题，内容很棒！')
      } else {
        message.info(`发现 ${result.corrections.length} 处需要修改的地方`)
      }
    } catch {
      // aiProofread already shows error toast via api/ai.ts
    } finally {
      setLoading(false)
    }
  }

  const handleInsertContent = () => {
    if (!editor) {
      message.warning('编辑器未就绪')
      return
    }
    editor.commands.insertContent(writeResult)
    message.success('内容已插入到编辑器')
  }

  const correctionTypeColors: Record<string, string> = {
    typo: 'red',
    grammar: 'orange',
    sensitive: 'volcano',
    style: 'blue',
  }
  const correctionTypeLabels: Record<string, string> = {
    typo: '错别字',
    grammar: '语法',
    sensitive: '敏感词',
    style: '风格',
  }

  if (!visible) return null

  return (
    <div
      style={{
        width: 380,
        padding: 16,
        borderLeft: '1px solid #f0f0f0',
        height: '100%',
        overflow: 'auto',
        background: '#fff',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Text strong style={{ fontSize: 16 }}>
          AI 助手
        </Text>
        <Button type="text" size="small" onClick={onClose}>
          关闭
        </Button>
      </div>

      <Spin spinning={loading}>
        <Tabs
          size="small"
          items={[
            {
              key: 'write',
              label: (
                <span>
                  <RobotOutlined /> 帮我写
                </span>
              ),
              children: (
                <div>
                  <Form<WriteFormValues>
                    layout="vertical"
                    onFinish={handleWrite}
                    size="small"
                    initialValues={{ style: '正式', max_words: 500 }}
                  >
                    <Form.Item
                      name="title"
                      label="文章主题"
                      rules={[{ required: true, message: '请输入文章主题' }]}
                    >
                      <Input placeholder="输入文章主题，如：公司第三季度业绩报告" />
                    </Form.Item>
                    <Form.Item name="keywords" label="关键词">
                      <Input placeholder="用逗号分隔关键词，如：业绩,增长,创新" />
                    </Form.Item>
                    <Form.Item name="style" label="写作风格">
                      <Select
                        options={[
                          { value: '正式', label: '正式' },
                          { value: '轻松', label: '轻松' },
                          { value: '活泼', label: '活泼' },
                          { value: '专业', label: '专业' },
                        ]}
                      />
                    </Form.Item>
                    <Form.Item name="max_words" label="字数限制">
                      <InputNumber
                        min={100}
                        max={2000}
                        step={100}
                        style={{ width: '100%' }}
                        placeholder="500"
                      />
                    </Form.Item>
                    <Form.Item>
                      <Button type="primary" htmlType="submit" loading={loading} block>
                        生成文章
                      </Button>
                    </Form.Item>
                  </Form>

                  {writeResult && (
                    <div
                      style={{
                        marginTop: 16,
                        padding: 12,
                        background: '#fafafa',
                        borderRadius: 8,
                        border: '1px solid #f0f0f0',
                      }}
                    >
                      <Text strong style={{ fontSize: 13 }}>
                        生成结果：
                      </Text>
                      <div
                        style={{ whiteSpace: 'pre-wrap', marginTop: 8, fontSize: 13, lineHeight: 1.8, color: '#333' }}
                      >
                        {writeResult}
                      </div>
                      <Button
                        type="link"
                        onClick={handleInsertContent}
                        style={{ padding: 0, marginTop: 8 }}
                      >
                        插入到编辑器
                      </Button>
                    </div>
                  )}
                </div>
              ),
            },
            {
              key: 'layout',
              label: (
                <span>
                  <FormatPainterOutlined /> 智能排版
                </span>
              ),
              children: (
                <div>
                  <p style={{ color: '#666', fontSize: 13 }}>
                    AI会分析当前内容结构，自动优化排版布局，包括标题层级、段落间距、色彩搭配等。
                  </p>

                  {!editor && (
                    <Alert
                      type="warning"
                      message="编辑器未连接"
                      description="请先打开编辑器再使用智能排版功能"
                      showIcon
                      style={{ marginTop: 12 }}
                    />
                  )}

                  <Button
                    type="primary"
                    onClick={handleLayout}
                    loading={loading}
                    disabled={!editor}
                    block
                    style={{ marginTop: 16 }}
                  >
                    开始智能排版
                  </Button>

                  {layoutResult && (
                    <Alert
                      type="success"
                      message="排版完成"
                      description="AI已优化页面布局，请预览查看效果。如需撤销，可使用编辑器撤销功能。"
                      showIcon
                      style={{ marginTop: 16 }}
                    />
                  )}
                </div>
              ),
            },
            {
              key: 'proofread',
              label: (
                <span>
                  <CheckCircleOutlined /> 智能校对
                </span>
              ),
              children: (
                <div>
                  <p style={{ color: '#666', fontSize: 13 }}>
                    检查错别字、语法错误、敏感词和风格问题。
                  </p>

                  {!editor && (
                    <Alert
                      type="warning"
                      message="编辑器未连接"
                      description="请先打开编辑器再使用智能校对功能"
                      showIcon
                      style={{ marginTop: 12 }}
                    />
                  )}

                  <Button
                    type="primary"
                    onClick={handleProofread}
                    loading={loading}
                    disabled={!editor}
                    block
                    style={{ marginTop: 16 }}
                  >
                    开始校对
                  </Button>

                  {proofreadDone && corrections.length > 0 && (
                    <List
                      style={{ marginTop: 16 }}
                      dataSource={corrections}
                      renderItem={(item, index) => (
                        <List.Item key={index} style={{ padding: '8px 0' }}>
                          <div style={{ width: '100%' }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                              <Tag color={correctionTypeColors[item.type] || 'default'}>
                                {correctionTypeLabels[item.type] || item.type}
                              </Tag>
                              <Text delete style={{ color: '#999' }}>
                                {item.original}
                              </Text>
                              <Text style={{ color: '#999' }}>→</Text>
                              <Text type="success" strong>
                                {item.suggestion}
                              </Text>
                            </div>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {item.explanation}
                            </Text>
                          </div>
                        </List.Item>
                      )}
                    />
                  )}

                  {proofreadDone && corrections.length === 0 && !loading && (
                    <div style={{ textAlign: 'center', padding: 32 }}>
                      <CheckCircleOutlined style={{ fontSize: 32, color: '#52c41a', marginBottom: 8 }} />
                      <div>
                        <Text type="secondary">未发现任何问题</Text>
                      </div>
                    </div>
                  )}

                  {!proofreadDone && !loading && (
                    <div style={{ textAlign: 'center', padding: 32 }}>
                      <Text type="secondary">点击上方按钮开始校对</Text>
                    </div>
                  )}
                </div>
              ),
            },
          ]}
        />
      </Spin>
    </div>
  )
}
