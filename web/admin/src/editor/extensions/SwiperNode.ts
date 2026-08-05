import { Node } from '@tiptap/core'

export interface SwiperImage {
  src: string
  alt: string
}

export interface SwiperAttrs {
  images: SwiperImage[]
  autoplay: boolean
  interval: number
}

export const SwiperNode = Node.create({
  name: 'swiperNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      images: {
        default: [] as SwiperImage[],
        parseHTML: (element: HTMLElement) => {
          const raw = element.getAttribute('data-images')
          if (!raw) return []
          try {
            return JSON.parse(raw) as SwiperImage[]
          } catch {
            return []
          }
        },
        renderHTML: (attrs: Record<string, unknown>) => {
          const images = attrs.images as SwiperImage[]
          return { images: JSON.stringify(images) }
        },
      },
      autoplay: {
        default: true,
        parseHTML: (element: HTMLElement) =>
          element.getAttribute('data-autoplay') !== 'false',
        renderHTML: (attrs: Record<string, unknown>) => ({
          autoplay: (attrs.autoplay as boolean).toString(),
        }),
      },
      interval: {
        default: 3000,
        parseHTML: (element: HTMLElement) => {
          const val = parseInt(element.getAttribute('data-interval') ?? '', 10)
          return Number.isNaN(val) ? 3000 : val
        },
        renderHTML: (attrs: Record<string, unknown>) => ({
          interval: (attrs.interval as number).toString(),
        }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="swiper"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { images, autoplay, interval } = HTMLAttributes as unknown as {
      images: string
      autoplay: string
      interval: string
    }
    let parsedImages: SwiperImage[] = []
    try {
      parsedImages = typeof images === 'string' ? JSON.parse(images) : (images as unknown as SwiperImage[])
    } catch {
      parsedImages = []
    }

    const firstImg = parsedImages.length > 0 ? parsedImages[0] : null

    return [
      'div',
      {
        'data-type': 'swiper',
        'data-images': typeof images === 'string' ? images : JSON.stringify(images),
        'data-autoplay': autoplay,
        'data-interval': interval,
        style: 'overflow:hidden;border-radius:8px;position:relative',
        class: 'swiper-node-wrapper',
      },
      firstImg
        ? [
            'img',
            {
              src: firstImg.src,
              alt: firstImg.alt,
              style: 'width:100%;display:block',
            },
          ]
        : [
            'div',
            {
              style:
                'background:#f0f0f0;height:200px;display:flex;align-items:center;justify-content:center;color:#999',
            },
            `轮播图 (${parsedImages.length} 张)`,
          ],
    ]
  },
})
