import { useCallback, useMemo, useState } from 'react';
import {
  DndContext,
  DragEndEvent,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import { Layout, Button, Tabs, Space, Tooltip, message, theme } from 'antd';
import {
  UndoOutlined,
  RedoOutlined,
  SaveOutlined,
  RobotOutlined,
} from '@ant-design/icons';
import { useEditor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import { ComponentPanel } from './ComponentPanel';
import { EditorCanvas } from './EditorCanvas';
import { PropertyPanel } from './PropertyPanel';
import { MobilePreview } from './MobilePreview';
import { TitleNode } from './extensions/TitleNode';
import { ParagraphNode } from './extensions/ParagraphNode';

const { Sider, Content } = Layout;
const { useToken } = theme;

const NODE_DEFAULTS: Record<string, Record<string, unknown>> = {
  titleNode: {
    level: 2,
    text: '请输入标题',
    color: '#333333',
    align: 'center',
    fontSize: 24,
  },
  paragraphNode: {
    color: '#333333',
    fontSize: 14,
    lineHeight: 1.8,
    align: 'left',
    spacing: 10,
  },
};

// Fallback default for node types without specific defaults
const FALLBACK_NODE_DEFAULTS: Record<string, unknown> = {
  color: '#333333',
  fontSize: 14,
  lineHeight: 1.8,
  align: 'left',
  spacing: 10,
};

function getNodeDefaults(type: string): Record<string, unknown> {
  return NODE_DEFAULTS[type] || FALLBACK_NODE_DEFAULTS;
}

// Map component type to the actual TipTap node type to insert
function mapToNodeType(componentType: string): string {
  const supportedTypes = ['titleNode', 'paragraphNode'];
  if (supportedTypes.includes(componentType)) {
    return componentType;
  }
  return 'paragraphNode';
}

const CUSTOM_EXTENSIONS = [StarterKit, TitleNode, ParagraphNode];

export function EditorPage() {
  const { token } = useToken();
  const [messageApi, contextHolder] = message.useMessage();
  const [activeDragType, setActiveDragType] = useState<string | null>(null);

  const editor = useEditor({
    extensions: CUSTOM_EXTENSIONS,
    content: {
      type: 'doc',
      content: [],
    },
    editorProps: {
      attributes: {
        style: `outline: none; min-height: 560px;`,
      },
    },
  });

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
    useSensor(KeyboardSensor),
  );

  const handleDragStart = useCallback(
    (event: { active: { id: string | number } }) => {
      const id = String(event.active.id);
      if (id.startsWith('new-')) {
        setActiveDragType(id.replace('new-', ''));
      }
    },
    [],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      setActiveDragType(null);

      if (!over || over.id !== 'editor-canvas') return;
      if (!editor) return;

      const componentType = String(active.id).replace('new-', '');
      const nodeType = mapToNodeType(componentType);
      const defaults = getNodeDefaults(componentType);

      if (componentType !== nodeType) {
        messageApi.info(`${componentType} 组件暂未适配，以段落形式插入`);
      }

      editor
        .chain()
        .focus()
        .insertContent({
          type: nodeType,
          attrs: defaults,
        })
        .run();
    },
    [editor, messageApi],
  );

  const handleUndo = useCallback(() => {
    editor?.chain().focus().undo().run();
  }, [editor]);

  const handleRedo = useCallback(() => {
    editor?.chain().focus().redo().run();
  }, [editor]);

  const handleSaveDraft = useCallback(() => {
    messageApi.success('草稿已保存');
  }, [messageApi]);

  const handleAIPanel = useCallback(() => {
    messageApi.info('AI 面板功能开发中');
  }, [messageApi]);

  const siderStyle: React.CSSProperties = useMemo(
    () => ({
      backgroundColor: token.colorBgContainer,
      borderRight: `1px solid ${token.colorBorderSecondary}`,
      borderLeft: `1px solid ${token.colorBorderSecondary}`,
      overflow: 'auto',
      height: '100%',
    }),
    [token],
  );

  const toolbarStyle: React.CSSProperties = useMemo(
    () => ({
      padding: '8px 16px',
      borderBottom: `1px solid ${token.colorBorderSecondary}`,
      backgroundColor: token.colorBgContainer,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      flexShrink: 0,
    }),
    [token],
  );

  const contentStyle: React.CSSProperties = useMemo(
    () => ({
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      overflow: 'auto',
    }),
    [],
  );

  const mainLayoutStyle: React.CSSProperties = useMemo(
    () => ({
      height: '100vh',
      backgroundColor: token.colorBgLayout,
    }),
    [token],
  );

  const tabItems = useMemo(
    () => [
      {
        key: 'edit',
        label: '编辑',
        children: <EditorCanvas editor={editor} />,
      },
      {
        key: 'preview',
        label: '手机预览',
        children: <MobilePreview editor={editor} />,
      },
    ],
    [editor],
  );

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      {contextHolder}

      <Layout style={mainLayoutStyle} data-testid="editor-page">
        <Sider width={220} style={siderStyle} data-testid="left-sider">
          <ComponentPanel />
        </Sider>

        <Content style={contentStyle}>
          <div style={toolbarStyle} data-testid="editor-toolbar">
            <Space>
              <Tooltip title="撤销 (Ctrl+Z)">
                <Button
                  icon={<UndoOutlined />}
                  onClick={handleUndo}
                  disabled={!editor?.can().undo()}
                  data-testid="undo-button"
                >
                  撤销
                </Button>
              </Tooltip>
              <Tooltip title="重做 (Ctrl+Y)">
                <Button
                  icon={<RedoOutlined />}
                  onClick={handleRedo}
                  disabled={!editor?.can().redo()}
                  data-testid="redo-button"
                >
                  重做
                </Button>
              </Tooltip>
            </Space>

            <Space>
              <Tooltip title="保存草稿">
                <Button
                  icon={<SaveOutlined />}
                  onClick={handleSaveDraft}
                  data-testid="save-draft-button"
                >
                  保存草稿
                </Button>
              </Tooltip>
              <Tooltip title="AI 智能助手">
                <Button
                  icon={<RobotOutlined />}
                  onClick={handleAIPanel}
                  type="primary"
                  ghost
                  data-testid="ai-panel-button"
                >
                  AI面板
                </Button>
              </Tooltip>
            </Space>
          </div>

          <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
            <Tabs
              items={tabItems}
              style={{ height: '100%' }}
              data-testid="editor-tabs"
            />
          </div>
        </Content>

        <Sider width={260} style={siderStyle} data-testid="right-sider">
          <PropertyPanel editor={editor} />
        </Sider>
      </Layout>

      <DragOverlay>
        {activeDragType ? (
          <div
            style={{
              padding: '8px 16px',
              backgroundColor: '#fff',
              border: '1px solid #1677ff',
              borderRadius: 6,
              boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
              fontSize: 14,
            }}
          >
            {activeDragType}
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}
