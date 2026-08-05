import { Node } from '@tiptap/core'

export interface ColumnItem {
  content: string
}

export interface ColumnsAttrs {
  count: 2 | 3
  gap: number
  items: ColumnItem[]
}

export const ColumnsNode = Node.create({
  name: 'columnsNode',
  group: 'block',
  content: 'block*',
  draggable: true,

  addAttributes() {
    return {
      count: {
        default: 2,
        parseHTML: (element: HTMLElement) => {
          const val = parseInt(element.getAttribute('data-count') ?? '', 10)
          return val === 3 ? 3 : 2
        },
        renderHTML: (attrs: Record<string, unknown>) => ({
          count: ((attrs.count as number) || 2).toString(),
        }),
      },
      gap: {
        default: 16,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/gap:\s*(\d+)px/)
          return match ? parseInt(match[1], 10) : 16
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ gap: attrs.gap as number }),
      },
      items: {
        default: [] as ColumnItem[],
        parseHTML: (element: HTMLElement) => {
          const raw = element.getAttribute('data-items')
          if (!raw) return []
          try {
            return JSON.parse(raw) as ColumnItem[]
          } catch {
            return []
          }
        },
        renderHTML: (attrs: Record<string, unknown>) => {
          const items = attrs.items as ColumnItem[]
          return { items: JSON.stringify(items) }
        },
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="columns"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { count, gap } = HTMLAttributes as unknown as {
      count: string
      gap: number
      items: string
    }
    const colCount = count === '3' ? 3 : 2
    return [
      'div',
      {
        'data-type': 'columns',
        'data-count': colCount.toString(),
        style: `display:flex;gap:${gap}px`,
        class: 'columns-node-wrapper',
      },
      0,
    ]
  },
})
