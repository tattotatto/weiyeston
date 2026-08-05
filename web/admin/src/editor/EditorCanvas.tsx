import { useDroppable } from '@dnd-kit/core';
import { EditorContent } from '@tiptap/react';
import { Empty } from 'antd';
import type { Editor } from '@tiptap/core';

interface EditorCanvasProps {
  editor: Editor | null;
}

export function EditorCanvas({ editor }: EditorCanvasProps) {
  const { setNodeRef, isOver } = useDroppable({ id: 'editor-canvas' });

  const canvasStyle: React.CSSProperties = {
    minHeight: 600,
    padding: 24,
    backgroundColor: isOver ? '#f0f5ff' : '#ffffff',
    border: isOver ? '2px dashed #1677ff' : '2px dashed #e8e8e8',
    borderRadius: 8,
    transition: 'border-color 0.2s, background-color 0.2s',
    position: 'relative',
    overflow: 'auto',
  };

  const emptyStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 600,
  };

  return (
    <div ref={setNodeRef} style={canvasStyle} data-testid="editor-canvas">
      {editor ? (
        <EditorContent
          editor={editor}
          style={{ minHeight: 560 }}
          data-testid="editor-content"
        />
      ) : (
        <div style={emptyStyle}>
          <Empty description="编辑器加载中..." />
        </div>
      )}
    </div>
  );
}
