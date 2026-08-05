import { Node } from '@tiptap/core'

export interface TitleAttrs {
  level: 1 | 2 | 3
  text: string
  color: string
  align: 'left' | 'center' | 'right'
  fontSize: number
}

export const TitleNode = Node.create({
  name: 'titleNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      level: {
        default: 2,
        parseHTML: (element: HTMLElement) => {
          const tag = element.tagName.toLowerCase()
          if (tag === 'h1') return 1
          if (tag === 'h3') return 3
          return 2
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ level: attrs.level as number }),
      },
      text: {
        default: '请输入标题',
        parseHTML: (element: HTMLElement) => element.textContent ?? '请输入标题',
        renderHTML: (attrs: Record<string, unknown>) => ({ text: attrs.text as string }),
      },
      color: {
        default: '#333333',
        parseHTML: (element: HTMLElement) => element.style.color || '#333333',
        renderHTML: (attrs: Record<string, unknown>) => ({ color: attrs.color as string }),
      },
      align: {
        default: 'center',
        parseHTML: (element: HTMLElement) => {
          const align = element.style.textAlign
          if (align === 'left' || align === 'right') return align
          return 'center'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ align: attrs.align as string }),
      },
      fontSize: {
        default: 24,
        parseHTML: (element: HTMLElement) => {
          const size = parseInt(element.style.fontSize, 10)
          return Number.isNaN(size) ? 24 : size
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ fontSize: attrs.fontSize as number }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="title"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { level, text, color, align, fontSize } = HTMLAttributes as unknown as TitleAttrs
    const tag = `h${level}`
    return [
      'div',
      {
        'data-type': 'title',
        style: `text-align:${align};color:${color};font-size:${fontSize}px`,
      },
      [tag, { style: `margin:0;line-height:1.4` }, text],
    ]
  },
})
