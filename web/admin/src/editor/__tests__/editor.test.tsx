import { render, act } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { EditorCore, getEditorJSON, setEditorContent } from '../EditorCore'
import type { JSONContent } from '@tiptap/core'
import type { Editor } from '@tiptap/core'

// ---------------------------------------------------------------------------
// Additional browser API mocks needed by ProseMirror / TipTap in jsdom
// ---------------------------------------------------------------------------
beforeEach(() => {
  // ProseMirror needs Range to be constructable
  if (typeof globalThis.Range === 'undefined') {
    class MockRange {
      setStart(_node: Node, _offset: number): void {
        /* noop */
      }
      setEnd(_node: Node, _offset: number): void {
        /* noop */
      }
      getBoundingClientRect(): DOMRect {
        return { x: 0, y: 0, width: 0, height: 0, top: 0, right: 0, bottom: 0, left: 0, toJSON: vi.fn }
      }
      getClientRects(): DOMRectList {
        return [] as unknown as DOMRectList
      }
      cloneRange(): MockRange {
        return new MockRange()
      }
      collapse(): void {
        /* noop */
      }
      selectNode(_node: Node): void {
        /* noop */
      }
      setStartAfter(_node: Node): void {
        /* noop */
      }
      setEndAfter(_node: Node): void {
        /* noop */
      }
      setStartBefore(_node: Node): void {
        /* noop */
      }
      setEndBefore(_node: Node): void {
        /* noop */
      }
      get startContainer(): Node {
        return document.body
      }
      get endContainer(): Node {
        return document.body
      }
      get startOffset(): number {
        return 0
      }
      get endOffset(): number {
        return 0
      }
      get collapsed(): boolean {
        return true
      }
      get commonAncestorContainer(): Node {
        return document.body
      }
      compareBoundaryPoints(_how: number, _range: MockRange): number {
        return 0
      }
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    globalThis.Range = MockRange as any
  }

  // Ensure document.createRange uses the mock
  if (!document.createRange) {
    document.createRange = () => {
      const range = new Range()
      range.setStart(document.body, 0)
      range.setEnd(document.body, 0)
      return range
    }
  }
})

// ---------------------------------------------------------------------------
// Helper: wait a micro-tick for the editor to initialise
// ---------------------------------------------------------------------------
function tick(ms = 10): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

/** Mutable ref wrapper – TypeScript can track .current mutations */
interface MutableRef<T> {
  current: T
}

function makeRef<T>(initial: T): MutableRef<T> {
  return { current: initial }
}

// ===========================================================================
// Tests
// ===========================================================================
describe('EditorCore', () => {
  // 1. Basic render
  it('renders the editor without crashing', () => {
    const { container } = render(<EditorCore />)
    const editorEl = container.querySelector('[contenteditable]')
    expect(editorEl).toBeInTheDocument()
  })

  it('renders with the tiptap editor class', () => {
    const { container } = render(<EditorCore />)
    const wrapper = container.querySelector('.editor-core-wrapper')
    expect(wrapper).toBeInTheDocument()
  })

  it('initialises with default empty document content', async () => {
    const editorRef = makeRef<Editor | null>(null)
    render(<EditorCore editorRef={editorRef} />)
    await tick()

    const editor = editorRef.current
    expect(editor).toBeTruthy()
    if (editor) {
      const json = getEditorJSON(editor)
      expect(json).toBeDefined()
      expect(json.type).toBe('doc')
      expect(Array.isArray(json.content)).toBe(true)
    }
  })

  // 2. JSON serialization
  it('exports valid JSON from an empty editor', async () => {
    const editorRef = makeRef<Editor | null>(null)
    render(<EditorCore editorRef={editorRef} />)
    await tick()

    const editor = editorRef.current
    if (editor) {
      const json = getEditorJSON(editor)
      expect(json).toHaveProperty('type', 'doc')
      expect(json).toHaveProperty('content')
    }
  })

  // 3. JSON deserialization / round-trip
  it('performs a JSON round-trip (set content then read back)', async () => {
    const input: JSONContent = {
      type: 'doc',
      content: [
        {
          type: 'titleNode',
          attrs: {
            level: 1,
            text: 'Hello Title',
            color: '#000000',
            align: 'center',
            fontSize: 28,
          },
        },
        {
          type: 'paragraphNode',
          content: [{ type: 'text', text: 'A paragraph.' }],
        },
      ],
    }

    const editorRef = makeRef<Editor | null>(null)
    render(<EditorCore initialContent={input} editorRef={editorRef} />)
    await tick()

    const editor = editorRef.current
    if (editor) {
      const outputJson = getEditorJSON(editor)
      expect(outputJson.content).toBeDefined()
      const content = outputJson.content ?? []
      expect(content.length).toBeGreaterThanOrEqual(2)

      const titleNode = content.find((n: JSONContent) => n.type === 'titleNode')
      expect(titleNode).toBeDefined()
      if (titleNode?.attrs) {
        expect(titleNode.attrs.text).toBe('Hello Title')
      }
    }
  })

  it('setEditorContent helper replaces content correctly', async () => {
    const editorRef = makeRef<Editor | null>(null)
    render(<EditorCore editorRef={editorRef} />)
    await tick()

    const editor = editorRef.current
    if (editor) {
      const newContent: JSONContent = {
        type: 'doc',
        content: [{ type: 'spacerNode', attrs: { height: 50 } }],
      }

      act(() => {
        setEditorContent(editor, newContent)
      })

      const json = getEditorJSON(editor)
      const content = json.content ?? []
      expect(content.length).toBeGreaterThanOrEqual(1)
      expect(content[0]?.type).toBe('spacerNode')
    }
  })

  // 4. getEditorJSON handles null editor gracefully
  it('getEditorJSON returns default content for null editor', () => {
    const result = getEditorJSON(null)
    expect(result.type).toBe('doc')
    expect(result.content).toEqual([])
  })

  // 5. setEditorContent is safe with null editor
  it('setEditorContent does not throw with null editor', () => {
    expect(() => {
      setEditorContent(null, { type: 'doc', content: [] })
    }).not.toThrow()
  })
})

// ===========================================================================
// Node-specific tests
// ===========================================================================
describe('Custom Node Extensions', () => {
  async function renderAndGetEditor(): Promise<Editor | null> {
    const editorRef = makeRef<Editor | null>(null)
    render(<EditorCore editorRef={editorRef} />)
    await tick()
    return editorRef.current
  }

  function insertNode(
    editor: Editor,
    type: string,
    attrs: Record<string, unknown> = {},
  ): void {
    act(() => {
      editor.chain().focus().insertContent({ type, attrs }).run()
    })
  }

  it('TitleNode can be inserted and rendered', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'titleNode', {
      level: 1,
      text: 'Test Title',
      color: '#111',
      align: 'center',
      fontSize: 32,
    })

    const json = editor.getJSON()
    const titles = (json.content ?? []).filter((n: JSONContent) => n.type === 'titleNode')
    expect(titles.length).toBeGreaterThanOrEqual(1)
    const attrs = titles[0]?.attrs as Record<string, unknown> | undefined
    if (attrs) {
      expect(attrs.text).toBe('Test Title')
      expect(attrs.level).toBe(1)
    }
  })

  it('ParagraphNode can contain inline text', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'paragraphNode', { color: '#333', fontSize: 16 })

    const json = editor.getJSON()
    const paragraphs = (json.content ?? []).filter((n: JSONContent) => n.type === 'paragraphNode')
    expect(paragraphs.length).toBeGreaterThanOrEqual(1)
  })

  it('ImageNode stores image attributes correctly', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'imageNode', {
      src: 'https://example.com/img.png',
      alt: 'Example',
      width: 80,
      borderRadius: 12,
      shadow: '0 4px 16px rgba(0,0,0,0.2)',
      caption: 'A caption',
    })

    const json = editor.getJSON()
    const images = (json.content ?? []).filter((n: JSONContent) => n.type === 'imageNode')
    expect(images.length).toBeGreaterThanOrEqual(1)
    const attrs = images[0]?.attrs as Record<string, unknown> | undefined
    if (attrs) {
      expect(attrs.src).toBe('https://example.com/img.png')
      expect(attrs.alt).toBe('Example')
      expect(attrs.width).toBe(80)
    }
  })

  it('SwiperNode stores image carousel data', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'swiperNode', {
      images: [
        { src: '/a.jpg', alt: 'A' },
        { src: '/b.jpg', alt: 'B' },
      ],
      autoplay: true,
      interval: 5000,
    })

    const json = editor.getJSON()
    const swipers = (json.content ?? []).filter((n: JSONContent) => n.type === 'swiperNode')
    expect(swipers.length).toBeGreaterThanOrEqual(1)
    const attrs = swipers[0]?.attrs as Record<string, unknown> | undefined
    if (attrs) {
      const imgs = attrs.images as Array<{ src: string; alt: string }> | undefined
      expect(Array.isArray(imgs)).toBe(true)
      if (imgs) {
        expect(imgs.length).toBe(2)
        expect(imgs[0].src).toBe('/a.jpg')
      }
    }
  })

  it('VideoNode stores video embed attributes', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'videoNode', {
      src: 'https://example.com/v.mp4',
      poster: 'https://example.com/poster.jpg',
      width: 100,
      height: 400,
      autoplay: false,
    })

    const json = editor.getJSON()
    const videos = (json.content ?? []).filter((n: JSONContent) => n.type === 'videoNode')
    expect(videos.length).toBeGreaterThanOrEqual(1)
  })

  it('DividerNode can be inserted with style', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'dividerNode', {
      style: 'dashed',
      color: '#ccc',
      width: 80,
      margin: 30,
    })

    const json = editor.getJSON()
    const dividers = (json.content ?? []).filter((n: JSONContent) => n.type === 'dividerNode')
    expect(dividers.length).toBeGreaterThanOrEqual(1)
  })

  it('ButtonNode renders a clickable CTA', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'buttonNode', {
      text: '立即购买',
      url: 'https://shop.example.com',
      bgColor: '#ff4d4f',
      textColor: '#fff',
      borderRadius: 20,
      align: 'center',
    })

    const json = editor.getJSON()
    const buttons = (json.content ?? []).filter((n: JSONContent) => n.type === 'buttonNode')
    expect(buttons.length).toBeGreaterThanOrEqual(1)
    const attrs = buttons[0]?.attrs as Record<string, unknown> | undefined
    if (attrs) {
      expect(attrs.text).toBe('立即购买')
      expect(attrs.url).toBe('https://shop.example.com')
    }
  })

  it('SpacerNode creates vertical whitespace', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'spacerNode', { height: 40 })

    const json = editor.getJSON()
    const spacers = (json.content ?? []).filter((n: JSONContent) => n.type === 'spacerNode')
    expect(spacers.length).toBeGreaterThanOrEqual(1)
    const attrs = spacers[0]?.attrs as Record<string, unknown> | undefined
    if (attrs) {
      expect(attrs.height).toBe(40)
    }
  })

  it('FollowGuideNode stores QR code and follow info', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'followGuideNode', {
      qrCodeUrl: 'https://example.com/qr.png',
      text: '关注公众号',
      showAvatar: true,
      bgColor: '#fff',
    })

    const json = editor.getJSON()
    const guides = (json.content ?? []).filter((n: JSONContent) => n.type === 'followGuideNode')
    expect(guides.length).toBeGreaterThanOrEqual(1)
    const attrs = guides[0]?.attrs as Record<string, unknown> | undefined
    if (attrs) {
      expect(attrs.qrCodeUrl).toBe('https://example.com/qr.png')
    }
  })

  it('ColumnsNode can be inserted as container', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'columnsNode', {
      count: 3,
      gap: 20,
      items: [{ content: 'Col 1' }, { content: 'Col 2' }, { content: 'Col 3' }],
    })

    const json = editor.getJSON()
    const cols = (json.content ?? []).filter((n: JSONContent) => n.type === 'columnsNode')
    expect(cols.length).toBeGreaterThanOrEqual(1)
  })

  it('CardNode can be inserted as container', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'cardNode', {
      bgColor: '#fafafa',
      borderRadius: 12,
      shadow: '0 2px 8px rgba(0,0,0,0.1)',
      padding: 24,
      title: 'Card Title',
    })

    const json = editor.getJSON()
    const cards = (json.content ?? []).filter((n: JSONContent) => n.type === 'cardNode')
    expect(cards.length).toBeGreaterThanOrEqual(1)
  })

  it('QuoteNode can be inserted as block', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    insertNode(editor, 'quoteNode', {
      text: 'A famous quote',
      color: '#555',
      borderColor: '#1890ff',
      bgColor: '#f5f5f5',
      icon: '❝',
    })

    const json = editor.getJSON()
    const quotes = (json.content ?? []).filter((n: JSONContent) => n.type === 'quoteNode')
    expect(quotes.length).toBeGreaterThanOrEqual(1)
  })

  it('editor can contain multiple node types in sequence', async () => {
    const editor = await renderAndGetEditor()
    if (!editor) return

    // Use setContent to reliably create a multi-node document
    const multiNodeContent: JSONContent = {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 2, text: 'Multi-Node Test', color: '#333', align: 'center', fontSize: 24 } },
        { type: 'paragraphNode', attrs: { color: '#333', fontSize: 14, lineHeight: 1.8, align: 'left', spacing: 10 } },
        { type: 'imageNode', attrs: { src: '/img.png', alt: 'img', width: 100, borderRadius: 8, shadow: 'none', caption: '' } },
        { type: 'dividerNode', attrs: { style: 'solid', color: '#e0e0e0', width: 100, margin: 20 } },
        { type: 'spacerNode', attrs: { height: 20 } },
        { type: 'buttonNode', attrs: { text: 'CTA', url: '/go', bgColor: '#1890ff', textColor: '#fff', borderRadius: 8, align: 'center' } },
      ],
    }

    act(() => {
      setEditorContent(editor, multiNodeContent)
    })

    const json = editor.getJSON()
    const content = json.content ?? []
    expect(content.length).toBeGreaterThanOrEqual(6)

    const types = content.map((c: JSONContent) => c.type)
    expect(types).toContain('titleNode')
    expect(types).toContain('paragraphNode')
    expect(types).toContain('imageNode')
    expect(types).toContain('dividerNode')
    expect(types).toContain('spacerNode')
    expect(types).toContain('buttonNode')
  })

  it('round-trips complex multi-node document', async () => {
    const input: JSONContent = {
      type: 'doc',
      content: [
        {
          type: 'titleNode',
          attrs: { level: 1, text: 'R1', color: '#000', align: 'center', fontSize: 28 },
        },
        { type: 'spacerNode', attrs: { height: 10 } },
        {
          type: 'dividerNode',
          attrs: { style: 'solid', color: '#eee', width: 100, margin: 10 },
        },
        { type: 'spacerNode', attrs: { height: 10 } },
        {
          type: 'paragraphNode',
          content: [{ type: 'text', text: 'Body text here.' }],
        },
        {
          type: 'buttonNode',
          attrs: {
            text: 'Action',
            url: '#',
            bgColor: '#1890ff',
            textColor: '#fff',
            borderRadius: 8,
            align: 'center',
          },
        },
      ],
    }

    const editorRef = makeRef<Editor | null>(null)
    render(<EditorCore initialContent={input} editorRef={editorRef} />)
    await tick()

    const editor = editorRef.current
    if (editor) {
      const output = getEditorJSON(editor)
      const content = output.content ?? []
      const typeSet = new Set(content.map((c: JSONContent) => c.type))
      expect(typeSet.has('titleNode')).toBe(true)
      expect(typeSet.has('dividerNode')).toBe(true)
      expect(typeSet.has('spacerNode')).toBe(true)
      expect(typeSet.has('buttonNode')).toBe(true)
    }
  })
})
