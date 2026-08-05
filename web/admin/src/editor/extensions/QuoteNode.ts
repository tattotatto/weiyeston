import { Node } from '@tiptap/core'

export interface QuoteAttrs {
  text: string
  color: string
  borderColor: string
  bgColor: string
  icon: string
}

export const QuoteNode = Node.create({
  name: 'quoteNode',
  group: 'block',
  content: 'inline*',
  draggable: true,

  addAttributes() {
    return {
      text: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const content = element.querySelector('[data-quote-text]')
          return content?.textContent ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ text: attrs.text as string }),
      },
      color: {
        default: '#666666',
        parseHTML: (element: HTMLElement) => {
          const content = element.querySelector('[data-quote-text]') as HTMLElement | null
          return content?.style.color || '#666666'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ color: attrs.color as string }),
      },
      borderColor: {
        default: '#1890ff',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/border-left-color:\s*([^;]+)/)
          return match ? match[1].trim() : '#1890ff'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ borderColor: attrs.borderColor as string }),
      },
      bgColor: {
        default: '#f9f9f9',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/background-color:\s*([^;]+)/)
          return match ? match[1].trim() : '#f9f9f9'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ bgColor: attrs.bgColor as string }),
      },
      icon: {
        default: '“',
        parseHTML: (element: HTMLElement) => {
          const iconEl = element.querySelector('[data-quote-icon]')
          return iconEl?.textContent ?? '“'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ icon: attrs.icon as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="quote"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { color, borderColor, bgColor, icon } =
      HTMLAttributes as unknown as QuoteAttrs
    return [
      'div',
      {
        'data-type': 'quote',
        style: `background-color:${bgColor};border-left:4px solid ${borderColor};padding:16px 20px;margin:10px 0;border-radius:0 8px 8px 0;position:relative`,
        class: 'quote-node-wrapper',
      },
      [
        'span',
        {
          'data-quote-icon': 'true',
          style: `position:absolute;top:4px;left:12px;font-size:32px;color:${borderColor};line-height:1;opacity:0.3`,
        },
        icon,
      ],
      [
        'div',
        {
          'data-quote-text': 'true',
          style: `color:${color};font-size:14px;line-height:1.8;padding-left:20px`,
        },
        0,
      ],
    ]
  },
})
