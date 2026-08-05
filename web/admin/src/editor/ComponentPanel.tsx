import { useDraggable } from '@dnd-kit/core';
import { Collapse, Typography } from 'antd';

const { Title } = Typography;

interface ComponentDef {
  type: string;
  label: string;
  icon: string;
  group: string;
}

const COMPONENTS: ComponentDef[] = [
  { type: 'titleNode', label: '标题', icon: '📝', group: '基础' },
  { type: 'paragraphNode', label: '段落', icon: '📄', group: '基础' },
  { type: 'quoteNode', label: '引用块', icon: '💬', group: '基础' },
  { type: 'imageNode', label: '单图', icon: '🖼️', group: '媒体' },
  { type: 'carouselNode', label: '轮播图', icon: '🎠', group: '媒体' },
  { type: 'videoNode', label: '视频', icon: '🎬', group: '媒体' },
  { type: 'dividerNode', label: '分隔线', icon: '➖', group: '布局' },
  { type: 'spacerNode', label: '留白', icon: '⬜', group: '布局' },
  { type: 'columnsNode', label: '多栏', icon: '📐', group: '布局' },
  { type: 'buttonNode', label: '按钮', icon: '🔘', group: '互动' },
  { type: 'cardNode', label: '卡片', icon: '🃏', group: '装饰' },
  { type: 'followGuideNode', label: '关注引导', icon: '➕', group: '装饰' },
];

const GROUP_ORDER = ['基础', '媒体', '布局', '互动', '装饰'] as const;

interface DraggableItemProps {
  type: string;
  label: string;
  icon: string;
}

function DraggableItem({ type, label, icon }: DraggableItemProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `new-${type}`,
  });

  const style: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    padding: '8px 12px',
    margin: '4px 0',
    border: '1px solid #d9d9d9',
    borderRadius: 6,
    cursor: 'grab',
    backgroundColor: isDragging ? '#e6f4ff' : '#fff',
    transition: 'background-color 0.2s, box-shadow 0.2s',
    boxShadow: isDragging ? '0 2px 8px rgba(0,0,0,0.15)' : 'none',
    zIndex: isDragging ? 1000 : 'auto',
    ...(transform
      ? {
          transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
        }
      : {}),
  };

  return (
    <div
      ref={setNodeRef}
      {...attributes}
      {...listeners}
      style={style}
      data-testid={`draggable-${type}`}
    >
      <span style={{ fontSize: 18 }}>{icon}</span>
      <span style={{ fontSize: 13, userSelect: 'none' }}>{label}</span>
    </div>
  );
}

export function ComponentPanel() {
  return (
    <div
      style={{ padding: '12px 8px', height: '100%', overflow: 'auto' }}
      data-testid="component-panel"
    >
      <Title level={5} style={{ textAlign: 'center', margin: '0 0 12px 0' }}>
        组件面板
      </Title>
      <Collapse
        defaultActiveKey={[...GROUP_ORDER]}
        size="small"
        items={GROUP_ORDER.map((group) => {
          const groupComponents = COMPONENTS.filter((c) => c.group === group);
          return {
            key: group,
            label: group,
            children: (
              <div data-testid={`component-group-${group}`}>
                {groupComponents.map((comp) => (
                  <DraggableItem
                    key={comp.type}
                    type={comp.type}
                    label={comp.label}
                    icon={comp.icon}
                  />
                ))}
              </div>
            ),
          };
        })}
      />
    </div>
  );
}
