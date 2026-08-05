import { Node } from '@tiptap/core'

export interface ButtonAttrs {
  text: string
  url: string
  bgColor: string
  textColor: string
  borderRadius: number
  align: 'left' | 'center' | 'right'
}

export const ButtonNode = Node.create({
  name: 'buttonNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      text: {
        default: '点击按钮',
        parseHTML: (element: HTMLElement) => {
          const btn = element.querySelector('a')
          return btn?.textContent ?? '点击按钮'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ text: attrs.text as string }),
      },
      url: {
        default: '#',
        parseHTML: (element: HTMLElement) => {
          const btn = element.querySelector('a')
          return btn?.getAttribute('href') ?? '#'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ url: attrs.url as string }),
      },
      bgColor: {
        default: '#1890ff',
        parseHTML: (element: HTMLElement) => {
          const btn = element.querySelector('a')
          return btn?.style.backgroundColor || '#1890ff'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ bgColor: attrs.bgColor as string }),
      },
      textColor: {
        default: '#ffffff',
        parseHTML: (element: HTMLElement) => {
          const btn = element.querySelector('a')
          return btn?.style.color || '#ffffff'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ textColor: attrs.textColor as string }),
      },
      borderRadius: {
        default: 8,
        parseHTML: (element: HTMLElement) => {
          const btn = element.querySelector('a')
          if (!btn) return 8
          const br = parseInt(btn.style.borderRadius, 10)
          return Number.isNaN(br) ? 8 : br
        },
        renderHTML: (attrs: Record<string, unknown>) => ({
          borderRadius: attrs.borderRadius as number,
        }),
      },
      align: {
        default: 'center',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          if (style.includes('text-align:left')) return 'left'
          if (style.includes('text-align:right')) return 'right'
          return 'center'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ align: attrs.align as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="button"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { text, url, bgColor, textColor, borderRadius, align } =
      HTMLAttributes as unknown as ButtonAttrs
    return [
      'div',
      {
        'data-type': 'button',
        style: `text-align:${align};padding:10px 0`,
        class: 'button-node-wrapper',
      },
      [
        'a',
        {
          href: url,
          target: '_blank',
          rel: 'noopener noreferrer',
          style: `display:inline-block;padding:10px 32px;background-color:${bgColor};color:${textColor};border-radius:${borderRadius}px;text-decoration:none;font-size:14px;cursor:pointer`,
        },
        text,
      ],
    ]
  },
})
