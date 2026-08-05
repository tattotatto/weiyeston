import { Node } from '@tiptap/core'

export interface ImageAttrs {
  src: string
  alt: string
  width: number
  borderRadius: number
  shadow: string
  caption: string
}

export const ImageNode = Node.create({
  name: 'imageNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const img = element.querySelector('img')
          return img?.getAttribute('src') ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ src: attrs.src as string }),
      },
      alt: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const img = element.querySelector('img')
          return img?.getAttribute('alt') ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ alt: attrs.alt as string }),
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
      borderRadius: {
        default: 8,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/border-radius:\s*(\d+)px/)
          return match ? parseInt(match[1], 10) : 8
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ borderRadius: attrs.borderRadius as number }),
      },
      shadow: {
        default: '0 2px 8px rgba(0,0,0,0.1)',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/box-shadow:\s*([^;]+)/)
          return match ? match[1].trim() : '0 2px 8px rgba(0,0,0,0.1)'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ shadow: attrs.shadow as string }),
      },
      caption: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const cap = element.querySelector('[data-caption]')
          return cap?.textContent ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ caption: attrs.caption as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="image"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { src, alt, width, borderRadius, shadow, caption } =
      HTMLAttributes as unknown as ImageAttrs
    return [
      'div',
      {
        'data-type': 'image',
        style: `text-align:center;padding:10px 0`,
        class: 'image-node-wrapper',
      },
      [
        'img',
        {
          src,
          alt,
          style: `width:${width}%;border-radius:${borderRadius}px;box-shadow:${shadow};max-width:100%;display:block;margin:0 auto`,
        },
      ],
      caption
        ? [
            'p',
            {
              'data-caption': 'true',
              style: 'text-align:center;color:#999;font-size:12px;margin-top:8px',
            },
            caption,
          ]
        : ['p', {}, ''],
    ]
  },
})
