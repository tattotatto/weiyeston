import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Form,
  Input,
  Select,
  InputNumber,
  Button,
  ColorPicker,
  Empty,
  Popconfirm,
  Typography,
  Divider,
} from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import type { Editor } from '@tiptap/core';

const { Title } = Typography;

interface PropertyPanelProps {
  editor: Editor | null;
}

interface FieldDef {
  name: string;
  label: string;
  fieldType: 'text' | 'number' | 'select' | 'color';
  options?: { label: string; value: string | number }[];
  min?: number;
  max?: number;
  step?: number;
}

const NODE_PROPERTIES: Record<string, FieldDef[]> = {
  titleNode: [
    { name: 'text', label: '文字内容', fieldType: 'text' },
    {
      name: 'level',
      label: '标题级别',
      fieldType: 'select',
      options: [
        { label: 'H1', value: 1 },
        { label: 'H2', value: 2 },
        { label: 'H3', value: 3 },
      ],
    },
    { name: 'color', label: '文字颜色', fieldType: 'color' },
    {
      name: 'align',
      label: '对齐方式',
      fieldType: 'select',
      options: [
        { label: '左对齐', value: 'left' },
        { label: '居中', value: 'center' },
        { label: '右对齐', value: 'right' },
      ],
    },
    { name: 'fontSize', label: '字体大小', fieldType: 'number', min: 12, max: 72, step: 1 },
  ],
  paragraphNode: [
    { name: 'color', label: '文字颜色', fieldType: 'color' },
    { name: 'fontSize', label: '字体大小', fieldType: 'number', min: 12, max: 36, step: 1 },
    { name: 'lineHeight', label: '行高', fieldType: 'number', min: 1, max: 3, step: 0.1 },
    {
      name: 'align',
      label: '对齐方式',
      fieldType: 'select',
      options: [
        { label: '左对齐', value: 'left' },
        { label: '居中', value: 'center' },
        { label: '右对齐', value: 'right' },
        { label: '两端对齐', value: 'justify' },
      ],
    },
    { name: 'spacing', label: '段落间距(px)', fieldType: 'number', min: 0, max: 60, step: 1 },
  ],
};

const CUSTOM_NODE_TYPES = [
  'titleNode',
  'paragraphNode',
  'quoteNode',
  'imageNode',
  'carouselNode',
  'videoNode',
  'dividerNode',
  'spacerNode',
  'columnsNode',
  'buttonNode',
  'cardNode',
  'followGuideNode',
];

interface ActiveNodeInfo {
  type: string;
  attrs: Record<string, unknown>;
}

function extractHexColor(colorValue: unknown): string {
  if (!colorValue) return '#333333';
  if (typeof colorValue === 'string') return colorValue;
  if (
    typeof colorValue === 'object' &&
    colorValue !== null &&
    'toHexString' in colorValue &&
    typeof (colorValue as { toHexString: () => string }).toHexString === 'function'
  ) {
    return (colorValue as { toHexString: () => string }).toHexString();
  }
  return '#333333';
}

function prepareFormValues(attrs: Record<string, unknown>): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(attrs)) {
    values[key] = value;
  }
  return values;
}

export function PropertyPanel({ editor }: PropertyPanelProps) {
  const [activeNode, setActiveNode] = useState<ActiveNodeInfo | null>(null);
  const [form] = Form.useForm();

  const propertyFields = useMemo(() => {
    if (!activeNode) return null;
    return NODE_PROPERTIES[activeNode.type] || null;
  }, [activeNode]);

  const scanSelection = useCallback(() => {
    if (!editor) {
      setActiveNode(null);
      return;
    }

    for (const nodeType of CUSTOM_NODE_TYPES) {
      if (editor.isActive(nodeType)) {
        const attrs = editor.getAttributes(nodeType);
        setActiveNode({ type: nodeType, attrs: attrs as Record<string, unknown> });

        const formValues = prepareFormValues(attrs as Record<string, unknown>);
        form.setFieldsValue(formValues);
        return;
      }
    }

    setActiveNode(null);
    form.resetFields();
  }, [editor, form]);

  useEffect(() => {
    if (!editor) return;

    editor.on('selectionUpdate', scanSelection);
    editor.on('update', scanSelection);
    scanSelection();

    return () => {
      editor.off('selectionUpdate', scanSelection);
      editor.off('update', scanSelection);
    };
  }, [editor, scanSelection]);

  const handleFieldChange = useCallback(
    (fieldName: string, value: unknown) => {
      if (!editor || !activeNode) return;

      let finalValue = value;
      if (fieldName === 'color') {
        finalValue = extractHexColor(value);
      }

      editor
        .chain()
        .focus()
        .updateAttributes(activeNode.type, { [fieldName]: finalValue })
        .run();

      setActiveNode((prev) => {
        if (!prev) return null;
        return {
          ...prev,
          attrs: { ...prev.attrs, [fieldName]: finalValue },
        };
      });
    },
    [editor, activeNode],
  );

  const handleDelete = useCallback(() => {
    if (!editor || !activeNode) return;

    editor.chain().focus().deleteNode(activeNode.type).run();
    setActiveNode(null);
    form.resetFields();
  }, [editor, activeNode, form]);

  const renderField = (field: FieldDef) => {
    const currentValue = activeNode?.attrs[field.name];

    switch (field.fieldType) {
      case 'text':
        return (
          <Form.Item key={field.name} name={field.name} label={field.label}>
            <Input
              value={(currentValue as string) ?? ''}
              onChange={(e) => handleFieldChange(field.name, e.target.value)}
              data-testid={`prop-field-${field.name}`}
            />
          </Form.Item>
        );

      case 'number':
        return (
          <Form.Item key={field.name} name={field.name} label={field.label}>
            <InputNumber
              value={(currentValue as number) ?? 0}
              min={field.min}
              max={field.max}
              step={field.step ?? 1}
              style={{ width: '100%' }}
              onChange={(val) => {
                if (val !== null) handleFieldChange(field.name, val);
              }}
              data-testid={`prop-field-${field.name}`}
            />
          </Form.Item>
        );

      case 'select':
        return (
          <Form.Item key={field.name} name={field.name} label={field.label}>
            <Select
              value={(currentValue as string | number) ?? field.options?.[0]?.value}
              options={field.options}
              onChange={(val) => handleFieldChange(field.name, val)}
              data-testid={`prop-field-${field.name}`}
            />
          </Form.Item>
        );

      case 'color':
        return (
          <Form.Item key={field.name} name={field.name} label={field.label}>
            <ColorPicker
              value={currentValue as string}
              onChange={(color) => handleFieldChange(field.name, color)}
              data-testid={`prop-field-${field.name}`}
            />
          </Form.Item>
        );

      default:
        return null;
    }
  };

  // Determine node type display name
  const nodeLabel = useMemo(() => {
    if (!activeNode) return '';
    const typeName: Record<string, string> = {
      titleNode: '标题',
      paragraphNode: '段落',
      quoteNode: '引用块',
      imageNode: '单图',
      carouselNode: '轮播图',
      videoNode: '视频',
      dividerNode: '分隔线',
      spacerNode: '留白',
      columnsNode: '多栏',
      buttonNode: '按钮',
      cardNode: '卡片',
      followGuideNode: '关注引导',
    };
    return typeName[activeNode.type] || activeNode.type;
  }, [activeNode]);

  return (
    <div
      style={{ padding: '12px 8px', height: '100%', overflow: 'auto' }}
      data-testid="property-panel"
    >
      <Title level={5} style={{ textAlign: 'center', margin: '0 0 12px 0' }}>
        属性面板
      </Title>

      {activeNode && propertyFields ? (
        <>
          <div
            style={{
              padding: '8px 12px',
              marginBottom: 12,
              backgroundColor: '#f5f5f5',
              borderRadius: 6,
              fontSize: 13,
              fontWeight: 500,
            }}
            data-testid="active-node-label"
          >
            当前组件：{nodeLabel}
          </div>

          <Form
            form={form}
            layout="vertical"
            size="small"
            data-testid="property-form"
          >
            {propertyFields.map(renderField)}
          </Form>

          <Divider style={{ margin: '16px 0 12px' }} />

          <Popconfirm
            title="确定删除此组件？"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={handleDelete}
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              block
              data-testid="delete-node-button"
            >
              删除组件
            </Button>
          </Popconfirm>
        </>
      ) : (
        <div style={{ padding: '40px 16px' }}>
          <Empty
            description="未选中组件"
            data-testid="no-selection-empty"
          />
          <div
            style={{
              textAlign: 'center',
              color: '#999',
              fontSize: 12,
              marginTop: 8,
            }}
          >
            点击画布中的组件以编辑属性
          </div>
        </div>
      )}
    </div>
  );
}
