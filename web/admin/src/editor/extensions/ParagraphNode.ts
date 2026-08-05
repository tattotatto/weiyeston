import { Node } from '@tiptap/core'

export interface ParagraphAttrs {
  color: string
  fontSize: number
  lineHeight: number
  align: 'left' | 'center' | 'right' | 'justify'
  spacing: number
}

export const ParagraphNode = Node.create({
  name: 'paragraphNode',
  group: 'block',
  content: 'inline*',
  draggable: true,

  addAttributes() {
    return {
      color: {
        default: '#333333',
        parseHTML: (element: HTMLElement) => element.style.color || '#333333',
        renderHTML: (attrs: Record<string, unknown>) => ({ color: attrs.color as string }),
      },
      fontSize: {
        default: 14,
        parseHTML: (element: HTMLElement) => {
          const size = parseInt(element.style.fontSize, 10)
          return Number.isNaN(size) ? 14 : size
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ fontSize: attrs.fontSize as number }),
      },
      lineHeight: {
        default: 1.8,
        parseHTML: (element: HTMLElement) => {
          const lh = parseFloat(element.style.lineHeight)
          return Number.isNaN(lh) ? 1.8 : lh
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ lineHeight: attrs.lineHeight as number }),
      },
      align: {
        default: 'left',
        parseHTML: (element: HTMLElement) => {
          const align = element.style.textAlign
          if (align === 'center' || align === 'right' || align === 'justify') return align
          return 'left'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ align: attrs.align as string }),
      },
      spacing: {
        default: 10,
        parseHTML: (element: HTMLElement) => {
          const margin = parseInt(element.style.marginBottom, 10)
          return Number.isNaN(margin) ? 10 : margin
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ spacing: attrs.spacing as number }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="paragraph"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { color, fontSize, lineHeight, align, spacing } =
      HTMLAttributes as unknown as ParagraphAttrs
    return [
      'div',
      {
        'data-type': 'paragraph',
        style: `color:${color};font-size:${fontSize}px;line-height:${lineHeight};text-align:${align};margin-bottom:${spacing}px`,
      },
      0,
    ]
  },
})
