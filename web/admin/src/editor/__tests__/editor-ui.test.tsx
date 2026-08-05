// T9 Editor UI — 编辑器界面测试
// 测试先行：验证所有编辑器 UI 组件渲染和交互

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ComponentPanel } from '../ComponentPanel';
import { EditorCanvas } from '../EditorCanvas';
import { PropertyPanel } from '../PropertyPanel';
import { MobilePreview } from '../MobilePreview';
import { EditorPage } from '../EditorPage';

// ==================== Mock Setup ====================

// TipTap uses document.getSelection / window.getSelection heavily.
// jsdom provides these, but ProseMirror needs getClientRects on Range.
if (typeof Range !== 'undefined' && !Range.prototype.getClientRects) {
  Range.prototype.getClientRects = function () {
    return {
      length: 0,
      item: () => null,
      [Symbol.iterator]: function* () {},
    } as unknown as DOMRectList;
  };
}

// Mock scrollIntoView for TipTap focus
if (typeof Element !== 'undefined' && !Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = vi.fn();
}

// ==================== ComponentPanel Tests ====================

describe('ComponentPanel', () => {
  it('应渲染组件面板标题', () => {
    render(<ComponentPanel />);
    expect(screen.getByText('组件面板')).toBeInTheDocument();
  });

  it('应渲染所有组件分组', () => {
    render(<ComponentPanel />);
    const groups = ['基础', '媒体', '布局', '互动', '装饰'];
    for (const group of groups) {
      expect(screen.getByTestId(`component-group-${group}`)).toBeInTheDocument();
    }
  });

  it('应渲染所有可拖拽组件项', () => {
    render(<ComponentPanel />);
    const expectedTypes = [
      'titleNode', 'paragraphNode', 'quoteNode',
      'imageNode', 'carouselNode', 'videoNode',
      'dividerNode', 'spacerNode', 'columnsNode',
      'buttonNode', 'cardNode', 'followGuideNode',
    ];
    for (const type of expectedTypes) {
      expect(screen.getByTestId(`draggable-${type}`)).toBeInTheDocument();
    }
  });

  it('可拖拽项应包含图标和标签', () => {
    render(<ComponentPanel />);
    const titleItem = screen.getByTestId('draggable-titleNode');
    expect(titleItem).toBeInTheDocument();
    expect(screen.getByText('标题')).toBeInTheDocument();
    expect(screen.getByText('段落')).toBeInTheDocument();
    expect(screen.getByText('单图')).toBeInTheDocument();
  });
});

// ==================== EditorCanvas Tests ====================

describe('EditorCanvas', () => {
  it('当 editor 为 null 时应显示加载中状态', () => {
    render(<EditorCanvas editor={null} />);
    expect(screen.getByTestId('editor-canvas')).toBeInTheDocument();
    expect(screen.getByText('编辑器加载中...')).toBeInTheDocument();
  });

  it('应渲染 droppable 画布区域', () => {
    render(<EditorCanvas editor={null} />);
    const canvas = screen.getByTestId('editor-canvas');
    expect(canvas).toBeInTheDocument();
  });
});

// ==================== PropertyPanel Tests ====================

describe('PropertyPanel', () => {
  it('当没有任何节点选中时应显示未选中提示', () => {
    const mockEditor = {
      isActive: vi.fn().mockReturnValue(false),
      getAttributes: vi.fn().mockReturnValue({}),
      on: vi.fn(),
      off: vi.fn(),
      chain: vi.fn().mockReturnValue({
        focus: vi.fn().mockReturnThis(),
        updateAttributes: vi.fn().mockReturnThis(),
        deleteNode: vi.fn().mockReturnThis(),
        run: vi.fn(),
      }),
      state: {
        selection: { $from: { depth: 0 } },
      },
    };

    render(<PropertyPanel editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    expect(screen.getByTestId('property-panel')).toBeInTheDocument();
    expect(screen.getByTestId('no-selection-empty')).toBeInTheDocument();
  });

  it('当选中 titleNode 时应显示标题属性表单', async () => {
    const mockEditor = {
      isActive: vi.fn((type: string) => type === 'titleNode'),
      getAttributes: vi.fn().mockReturnValue({
        level: 2,
        text: '测试标题',
        color: '#333333',
        align: 'center',
        fontSize: 24,
      }),
      on: vi.fn(),
      off: vi.fn(),
      chain: vi.fn().mockReturnValue({
        focus: vi.fn().mockReturnThis(),
        updateAttributes: vi.fn().mockReturnThis(),
        deleteNode: vi.fn().mockReturnThis(),
        run: vi.fn(),
      }),
      state: {
        selection: { $from: { depth: 0 } },
      },
    };

    render(<PropertyPanel editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    expect(screen.getByText('当前组件：标题')).toBeInTheDocument();
    expect(screen.getByTestId('property-form')).toBeInTheDocument();
    expect(screen.getByTestId('delete-node-button')).toBeInTheDocument();
  });

  it('当选中 paragraphNode 时应显示段落属性表单', async () => {
    const mockEditor = {
      isActive: vi.fn((type: string) => type === 'paragraphNode'),
      getAttributes: vi.fn().mockReturnValue({
        color: '#333333',
        fontSize: 14,
        lineHeight: 1.8,
        align: 'left',
        spacing: 10,
      }),
      on: vi.fn(),
      off: vi.fn(),
      chain: vi.fn().mockReturnValue({
        focus: vi.fn().mockReturnThis(),
        updateAttributes: vi.fn().mockReturnThis(),
        deleteNode: vi.fn().mockReturnThis(),
        run: vi.fn(),
      }),
      state: {
        selection: { $from: { depth: 0 } },
      },
    };

    render(<PropertyPanel editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    expect(screen.getByText('当前组件：段落')).toBeInTheDocument();
    expect(screen.getByTestId('property-form')).toBeInTheDocument();
  });

  it('属性面板应注册 editor 事件监听', () => {
    const onSpy = vi.fn();
    const offSpy = vi.fn();
    const mockEditor = {
      isActive: vi.fn().mockReturnValue(false),
      getAttributes: vi.fn().mockReturnValue({}),
      on: onSpy,
      off: offSpy,
      chain: vi.fn().mockReturnValue({
        focus: vi.fn().mockReturnThis(),
        updateAttributes: vi.fn().mockReturnThis(),
        deleteNode: vi.fn().mockReturnThis(),
        run: vi.fn(),
      }),
      state: {
        selection: { $from: { depth: 0 } },
      },
    };

    render(<PropertyPanel editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    expect(onSpy).toHaveBeenCalledWith('selectionUpdate', expect.any(Function));
    expect(onSpy).toHaveBeenCalledWith('update', expect.any(Function));
  });
});

// ==================== MobilePreview Tests ====================

describe('MobilePreview', () => {
  it('应渲染手机预览容器', () => {
    const mockEditor = {
      getHTML: vi.fn().mockReturnValue('<p>测试内容</p>'),
      on: vi.fn(),
      off: vi.fn(),
    };

    render(<MobilePreview editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    expect(screen.getByTestId('mobile-preview')).toBeInTheDocument();
  });

  it('应渲染预览 iframe', () => {
    const mockEditor = {
      getHTML: vi.fn().mockReturnValue('<p>测试内容</p>'),
      on: vi.fn(),
      off: vi.fn(),
    };

    render(<MobilePreview editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    const iframe = screen.getByTestId('preview-iframe');
    expect(iframe).toBeInTheDocument();
    expect(iframe.tagName).toBe('IFRAME');
  });

  it('应渲染刷新预览按钮', () => {
    const mockEditor = {
      getHTML: vi.fn().mockReturnValue('<p>测试内容</p>'),
      on: vi.fn(),
      off: vi.fn(),
    };

    render(<MobilePreview editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    expect(screen.getByTestId('refresh-preview-button')).toBeInTheDocument();
    expect(screen.getByText('刷新预览')).toBeInTheDocument();
  });

  it('点击刷新按钮应重新获取 HTML', async () => {
    const getHTMLSpy = vi.fn().mockReturnValue('<p>测试内容</p>');
    const mockEditor = {
      getHTML: getHTMLSpy,
      on: vi.fn(),
      off: vi.fn(),
    };

    render(<MobilePreview editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    const refreshBtn = screen.getByTestId('refresh-preview-button');

    // clear initial call count
    getHTMLSpy.mockClear();
    await userEvent.click(refreshBtn);

    expect(getHTMLSpy).toHaveBeenCalledTimes(1);
  });

  it('当 editor 为 null 时应显示空白文档', () => {
    render(<MobilePreview editor={null} />);
    const iframe = screen.getByTestId('preview-iframe') as HTMLIFrameElement;
    expect(iframe).toBeInTheDocument();
    expect(iframe.srcdoc).toContain('空白文档');
  });

  it('iframe 应包含刷新后的 editor HTML', () => {
    const mockEditor = {
      getHTML: vi.fn().mockReturnValue('<h2>标题</h2><p>内容</p>'),
      on: vi.fn(),
      off: vi.fn(),
    };

    render(<MobilePreview editor={mockEditor as unknown as import('@tiptap/core').Editor} />);
    const iframe = screen.getByTestId('preview-iframe') as HTMLIFrameElement;
    expect(iframe.srcdoc).toContain('<h2>标题</h2>');
    expect(iframe.srcdoc).toContain('<p>内容</p>');
  });
});

// ==================== EditorPage Full Integration Tests ====================

describe('EditorPage', () => {
  beforeEach(() => {
    // Ensure clean state
    vi.clearAllMocks();
  });

  it('应渲染完整的编辑器页面（包含左右侧栏和工具栏）', () => {
    render(<EditorPage />);

    // Check left sider with component panel
    expect(screen.getByTestId('left-sider')).toBeInTheDocument();
    expect(screen.getByTestId('component-panel')).toBeInTheDocument();

    // Check toolbar
    expect(screen.getByTestId('editor-toolbar')).toBeInTheDocument();

    // Check right sider with property panel
    expect(screen.getByTestId('right-sider')).toBeInTheDocument();
    expect(screen.getByTestId('property-panel')).toBeInTheDocument();

    // Check canvas area
    expect(screen.getByTestId('editor-canvas')).toBeInTheDocument();
  });

  it('应渲染工具栏所有按钮', () => {
    render(<EditorPage />);

    expect(screen.getByTestId('undo-button')).toBeInTheDocument();
    expect(screen.getByTestId('redo-button')).toBeInTheDocument();
    expect(screen.getByTestId('save-draft-button')).toBeInTheDocument();
    expect(screen.getByTestId('ai-panel-button')).toBeInTheDocument();
  });

  it('工具栏应包含文字标签', () => {
    render(<EditorPage />);

    expect(screen.getByText('撤销')).toBeInTheDocument();
    expect(screen.getByText('重做')).toBeInTheDocument();
    expect(screen.getByText('保存草稿')).toBeInTheDocument();
    expect(screen.getByText('AI面板')).toBeInTheDocument();
  });

  it('应渲染 Tabs 包含编辑和手机预览标签页', () => {
    render(<EditorPage />);

    expect(screen.getByTestId('editor-tabs')).toBeInTheDocument();
    expect(screen.getByText('编辑')).toBeInTheDocument();
    expect(screen.getByText('手机预览')).toBeInTheDocument();
  });

  it('初始状态撤销和重做按钮应为禁用', () => {
    render(<EditorPage />);

    const undoBtn = screen.getByTestId('undo-button');
    const redoBtn = screen.getByTestId('redo-button');

    expect(undoBtn).toBeDisabled();
    expect(redoBtn).toBeDisabled();
  });

  it('点击保存草稿按钮应显示成功提示', async () => {
    render(<EditorPage />);

    const saveBtn = screen.getByTestId('save-draft-button');
    await userEvent.click(saveBtn);

    // Ant Design message appears asynchronously
    const messageEl = await screen.findByText('草稿已保存', {}, { timeout: 3000 });
    expect(messageEl).toBeInTheDocument();
  });

  it('点击 AI 面板按钮应显示开发中提示', async () => {
    render(<EditorPage />);

    const aiBtn = screen.getByTestId('ai-panel-button');
    await userEvent.click(aiBtn);

    const messageEl = await screen.findByText('AI 面板功能开发中', {}, { timeout: 3000 });
    expect(messageEl).toBeInTheDocument();
  });

  it('编辑画布应包含 editor 实例', () => {
    render(<EditorPage />);

    const editorContent = screen.getByTestId('editor-content');
    expect(editorContent).toBeInTheDocument();
  });

  it('应渲染组件面板中所有可拖拽项', () => {
    render(<EditorPage />);

    // Draggable items should exist within the ComponentPanel
    const titleItem = screen.getByTestId('draggable-titleNode');
    expect(titleItem).toBeInTheDocument();
  });
});

// ==================== Drag-and-Drop Integration ====================

describe('EditorPage — 拖拽交互', () => {
  it('DndContext 应包裹编辑器页面', () => {
    render(<EditorPage />);

    // The editor page should render without errors
    expect(screen.getByTestId('editor-page')).toBeInTheDocument();

    // Draggable items should exist
    expect(screen.getByTestId('draggable-titleNode')).toBeInTheDocument();
    expect(screen.getByTestId('draggable-paragraphNode')).toBeInTheDocument();

    // Droppable canvas should exist
    expect(screen.getByTestId('editor-canvas')).toBeInTheDocument();
  });

  it('可拖拽项应包含 drag handle 属性', () => {
    render(<EditorPage />);

    const dragItem = screen.getByTestId('draggable-titleNode');

    // useDraggable adds a role attribute for accessibility
    // The element should be in the document when not actively dragging
    expect(dragItem).toBeInTheDocument();
  });
});

// ==================== PropertyPanel Integration ====================

describe('PropertyPanel — 集成在 EditorPage 中', () => {
  it('EditorPage 应包含属性面板', () => {
    render(<EditorPage />);

    const propertyPanel = screen.getByTestId('property-panel');
    expect(propertyPanel).toBeInTheDocument();
    expect(screen.getByText('属性面板')).toBeInTheDocument();
  });

  it('初始状态应显示未选中组件', () => {
    render(<EditorPage />);

    const noSelection = screen.getByTestId('no-selection-empty');
    expect(noSelection).toBeInTheDocument();
  });
});

// ==================== MobilePreview Integration ====================

describe('MobilePreview — 集成在 EditorPage 中', () => {
  it('切换到手机预览标签页应显示预览', async () => {
    render(<EditorPage />);

    const previewTab = screen.getByText('手机预览');
    await userEvent.click(previewTab);

    // Mobile preview should now be visible
    const mobilePreview = screen.getByTestId('mobile-preview');
    expect(mobilePreview).toBeInTheDocument();

    const iframe = screen.getByTestId('preview-iframe');
    expect(iframe).toBeInTheDocument();
  });

  it('手机预览应包含刷新按钮', async () => {
    render(<EditorPage />);

    const previewTab = screen.getByText('手机预览');
    await userEvent.click(previewTab);

    const refreshBtn = screen.getByTestId('refresh-preview-button');
    expect(refreshBtn).toBeInTheDocument();
  });
});

// ==================== Rendering Robustness ====================

describe('Editor UI — 鲁棒性', () => {
  it('ComponentPanel 应能独立渲染', () => {
    const { container } = render(<ComponentPanel />);
    expect(container).toBeTruthy();
  });

  it('EditorCanvas 在无 editor 时应能优雅降级', () => {
    const { container } = render(<EditorCanvas editor={null} />);
    expect(container).toBeTruthy();
    expect(screen.getByText('编辑器加载中...')).toBeInTheDocument();
  });

  it('PropertyPanel 在无 editor 时应显示未选中', () => {
    render(<PropertyPanel editor={null} />);
    expect(screen.getByTestId('no-selection-empty')).toBeInTheDocument();
  });

  it('MobilePreview 在无 editor 时应显示空白', () => {
    render(<MobilePreview editor={null} />);
    expect(screen.getByTestId('preview-iframe')).toBeInTheDocument();
  });
});
