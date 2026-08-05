import { Node } from '@tiptap/core'

export interface FollowGuideAttrs {
  qrCodeUrl: string
  text: string
  showAvatar: boolean
  bgColor: string
}

export const FollowGuideNode = Node.create({
  name: 'followGuideNode',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      qrCodeUrl: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          const img = element.querySelector('img[data-qr]')
          return img?.getAttribute('src') ?? ''
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ qrCodeUrl: attrs.qrCodeUrl as string }),
      },
      text: {
        default: '扫码关注我们',
        parseHTML: (element: HTMLElement) => {
          const textEl = element.querySelector('[data-follow-text]')
          return textEl?.textContent ?? '扫码关注我们'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ text: attrs.text as string }),
      },
      showAvatar: {
        default: true,
        parseHTML: (element: HTMLElement) =>
          element.getAttribute('data-show-avatar') !== 'false',
        renderHTML: (attrs: Record<string, unknown>) => ({
          showAvatar: (attrs.showAvatar as boolean).toString(),
        }),
      },
      bgColor: {
        default: '#f5f5f5',
        parseHTML: (element: HTMLElement) => {
          const style = element.getAttribute('style') ?? ''
          const match = style.match(/background-color:\s*([^;]+)/)
          return match ? match[1].trim() : '#f5f5f5'
        },
        renderHTML: (attrs: Record<string, unknown>) => ({ bgColor: attrs.bgColor as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="followGuide"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const { qrCodeUrl, text, showAvatar, bgColor } =
      HTMLAttributes as unknown as FollowGuideAttrs
    return [
      'div',
      {
        'data-type': 'followGuide',
        'data-show-avatar': showAvatar.toString(),
        style: `background-color:${bgColor};border-radius:12px;padding:24px;text-align:center;margin:10px 0`,
        class: 'follow-guide-node-wrapper',
      },
      showAvatar
        ? [
            'div',
            {
              style:
                'width:60px;height:60px;border-radius:50%;background:#1890ff;color:#fff;display:flex;align-items:center;justify-content:center;font-size:24px;margin:0 auto 16px',
            },
            '👥',
          ]
        : ['div', {}, ''],
      qrCodeUrl
        ? [
            'img',
            {
              'data-qr': 'true',
              src: qrCodeUrl,
              alt: 'QR Code',
              style: 'width:160px;height:160px;display:block;margin:0 auto 12px;border-radius:8px',
            },
          ]
        : [
            'div',
            {
              style:
                'width:160px;height:160px;background:#e0e0e0;display:flex;align-items:center;justify-content:center;margin:0 auto 12px;border-radius:8px;color:#999;font-size:12px',
            },
            '二维码占位',
          ],
      [
        'p',
        {
          'data-follow-text': 'true',
          style: 'color:#666;font-size:14px;margin:0',
        },
        text,
      ],
    ]
  },
})
