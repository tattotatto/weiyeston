import { Node } from '@tiptap/core'

export interface SpacerAttrs {
  height: number
}

export const SpacerNode = Node.create({
  name: 'spacerNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      height: {
        default: 20,
        parseHTML: (element: HTMLElement) => {
          const height = parseInt(element.style.height, 10)
          return Number.isNaN(height) ? 20 : height
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ height: attrs.height as number }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="spacer"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { height } = HTMLAttributes as unknown as SpacerAttrs
    return [
      'div',
      {
        'data-type': 'spacer',
        style: `height:${height}px;width:100%`,
        class: 'spacer-node-wrapper',
      },
      '',
    ]
  },
})
