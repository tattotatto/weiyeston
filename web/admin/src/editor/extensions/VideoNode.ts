import { Node } from '@tiptap/core'

export interface VideoAttrs {
  src: string
  poster: string
  width: number
  height: number
  autoplay: boolean
}

export const VideoNode = Node.create({
  name: 'videoNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const video = element.querySelector('video')
          return video?.getAttribute('src') ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ src: attrs.src as string }),
      },
      poster: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const video = element.querySelector('video')
          return video?.getAttribute('poster') ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ poster: attrs.poster as string }),
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
      height: {
        default: 400,
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/height:\s*(\d+)px/)
          return match ? parseInt(match[1], 10) : 400
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ height: attrs.height as number }),
      },
      autoplay: {
        default: false,
        parseHTML: (element: HTMLElement) => {
          const video = element.querySelector('video')
          return video?.hasAttribute('autoplay') ?? false
        },
        renderHTML: (attrs: Record<string, unknown>) => ({
          autoplay: (attrs.autoplay as boolean).toString(),
        }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="video"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { src, poster, width, height, autoplay } =
      HTMLAttributes as unknown as VideoAttrs
    const autoplayStr = autoplay ? 'autoplay' : ''
    return [
      'div',
      {
        'data-type': 'video',
        style: `text-align:center;padding:10px 0`,
        class: 'video-node-wrapper',
      },
      [
        'video',
        {
          src,
          poster: poster || undefined,
          controls: 'true',
          [autoplayStr]: autoplay ? 'true' : undefined,
          style: `width:${width}%;height:${height}px;border-radius:8px;max-width:100%`,
        },
        '',
      ],
    ]
  },
})
