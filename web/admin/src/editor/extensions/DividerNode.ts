import { Node } from '@tiptap/core'

export type DividerStyle = 'solid' | 'dashed' | 'dotted' | 'gradient'

export interface DividerAttrs {
  style: DividerStyle
  color: string
  width: number
  margin: number
}

export const DividerNode = Node.create({
  name: 'dividerNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      style: {
        default: 'solid',
        parseHTML: (element: HTMLElement) => {
          const ds = element.getAttribute('data-divider-style')
          if (ds === 'dashed' || ds === 'dotted' || ds === 'gradient') return ds
          return 'solid'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({
          style: attrs.style as string,
        }),
      },
      color: {
        default: '#e0e0e0',
        parseHTML: (element: HTMLElement) => {
          const hr = element.querySelector('hr')
          if (!hr) return '#e0e0e0'
          const borderColor = hr.style.borderTopColor || hr.style.borderColor
          return borderColor || '#e0e0e0'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ color: attrs.color as string }),
      },
      width: {
        default: 100,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/width:\s*(\d+)%/)
          return match ? parseInt(match[1], 10) : 100
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ width: attrs.width as number }),
      },
      margin: {
        default: 20,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/margin:\s*(\d+)px\s+0/)
          return match ? parseInt(match[1], 10) : 20
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ margin: attrs.margin as number }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="divider"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { style, color, width, margin } =
      HTMLAttributes as unknown as DividerAttrs

    if (style === 'gradient') {
      return [
        'div',
        {
          'data-type': 'divider',
          'data-divider-style': 'gradient',
          style: `width:${width}%;margin:${margin}px auto`,
          class: 'divider-node-wrapper',
        },
        [
          'div',
          {
            style: `height:2px;background:linear-gradient(to right,transparent,${color},transparent)`,
          },
          '',
        ],
      ]
    }

    const borderStyle =
      style === 'dashed' ? 'dashed' : style === 'dotted' ? 'dotted' : 'solid'

    return [
      'div',
      {
        'data-type': 'divider',
        'data-divider-style': style,
        style: `width:${width}%;margin:${margin}px auto`,
        class: 'divider-node-wrapper',
      },
      [
        'hr',
        {
          style: `border:none;border-top:1px ${borderStyle} ${color}`,
        },
      ],
    ]
  },
})
