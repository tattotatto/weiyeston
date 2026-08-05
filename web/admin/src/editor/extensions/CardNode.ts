import { Node } from '@tiptap/core'

export interface CardAttrs {
  bgColor: string
  borderRadius: number
  shadow: string
  padding: number
  title: string
}

export const CardNode = Node.create({
  name: 'cardNode',
  group: 'block',
  content: 'block*',
  draggable: true,

  addAttributes() {
    return {
      bgColor: {
        default: '#ffffff',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/background-color:\s*([^;]+)/)
          return match ? match[1].trim() : '#ffffff'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ bgColor: attrs.bgColor as string }),
      },
      borderRadius: {
        default: 12,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/border-radius:\s*(\d+)px/)
          return match ? parseInt(match[1], 10) : 12
        },
        renderHTML: (attrs: Record<string, unknown>) => ({
          borderRadius: attrs.borderRadius as number,
        }),
      },
      shadow: {
        default: '0 2px 12px rgba(0,0,0,0.08)',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/box-shadow:\s*([^;]+)/)
          return match ? match[1].trim() : '0 2px 12px rgba(0,0,0,0.08)'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ shadow: attrs.shadow as string }),
      },
      padding: {
        default: 20,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/padding:\s*(\d+)px/)
          return match ? parseInt(match[1], 10) : 20
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ padding: attrs.padding as number }),
      },
      title: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const titleEl = element.querySelector('[data-card-title]')
          return titleEl?.textContent ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ title: attrs.title as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="card"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { bgColor, borderRadius, shadow, padding, title } =
      HTMLAttributes as unknown as CardAttrs

    // ProseMirror requires that 0 (content hole) is the ONLY child of its
    // immediate parent element. We wrap it in a dedicated content div.
    const specChildren: Array<string | Record<string, unknown> | Array<unknown>> = []

    if (title) {
      specChildren.push([
        'div',
        {
          'data-card-title': 'true',
          style: 'font-size:16px;font-weight:600;margin-bottom:12px;color:#333',
        },
        title,
      ])
    }

    specChildren.push([
      'div',
      {
        'data-card-content': 'true',
      },
      0,
    ])

    return [
      'div',
      {
        'data-type': 'card',
        style: `background-color:${bgColor};border-radius:${borderRadius}px;box-shadow:${shadow};padding:${padding}px`,
        class: 'card-node-wrapper',
      },
      ...specChildren,
    ]
  },
})
