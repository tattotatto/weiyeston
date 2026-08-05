export interface PresetTemplate {
  id: string
  name: string
  description: string
  category: string // 节日/活动/新闻
  coverUrl: string
  content: Record<string, unknown> // TipTap JSON content
  tags: string[]
}

export const PRESET_TEMPLATES: PresetTemplate[] = [
  {
    id: 'spring-festival',
    name: '春节喜庆',
    description: '红红火火的春节活动页面模板',
    category: '节日',
    coverUrl: '',
    content: {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 1, text: '新春快乐', color: '#FF0000', align: 'center', fontSize: 32 } },
        { type: 'paragraphNode', attrs: { color: '#333333', align: 'center', fontSize: 16, lineHeight: 1.8, spacing: 10 }, content: [{ type: 'text', text: '祝大家新春快乐，万事如意！' }] },
        { type: 'dividerNode', attrs: { style: 'solid', color: '#FF0000', width: 100, margin: 16 } },
        { type: 'imageNode', attrs: { src: '', alt: '春节图片', width: 750, borderRadius: 8, shadow: 'none', caption: '新春贺岁' } },
        { type: 'buttonNode', attrs: { text: '立即参与', url: '', bgColor: '#FF0000', textColor: '#FFFFFF', borderRadius: 4, align: 'center' } },
      ],
    },
    tags: ['春节', '红色', '喜庆'],
  },
  {
    id: 'mid-autumn',
    name: '中秋团圆',
    description: '温馨雅致的中秋祝福页面模板',
    category: '节日',
    coverUrl: '',
    content: {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 1, text: '中秋快乐', color: '#D4A017', align: 'center', fontSize: 30 } },
        { type: 'imageNode', attrs: { src: '', alt: '中秋月亮', width: 750, borderRadius: 16, shadow: '0 2px 8px rgba(0,0,0,0.1)', caption: '月圆人团圆' } },
        { type: 'paragraphNode', attrs: { color: '#555555', align: 'center', fontSize: 15, lineHeight: 2, spacing: 8 }, content: [{ type: 'text', text: '海上生明月，天涯共此时。祝您中秋快乐，阖家幸福！' }] },
        { type: 'dividerNode', attrs: { style: 'dashed', color: '#D4A017', width: 80, margin: 12 } },
        { type: 'cardNode', attrs: { bgColor: '#FFF8E1', borderRadius: 12, shadow: '0 2px 4px rgba(0,0,0,0.08)', padding: 24, title: '中秋活动' }, content: [{ type: 'paragraphNode', attrs: { color: '#333333', align: 'left', fontSize: 14, lineHeight: 1.6, spacing: 10 }, content: [{ type: 'text', text: '桂花飘香，月饼甜美。参与我们的中秋特别活动，赢取精美礼品！' }] }] },
        { type: 'buttonNode', attrs: { text: '查看活动详情', url: '', bgColor: '#D4A017', textColor: '#FFFFFF', borderRadius: 8, align: 'center' } },
      ],
    },
    tags: ['中秋', '团圆', '温馨'],
  },
  {
    id: 'promotion',
    name: '促销活动',
    description: '吸引力十足的促销活动页面模板',
    category: '活动',
    coverUrl: '',
    content: {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 1, text: '限时特惠', color: '#FF6600', align: 'center', fontSize: 36 } },
        { type: 'swiperNode', attrs: { images: [{ src: '', alt: '海报1' }, { src: '', alt: '海报2' }, { src: '', alt: '海报3' }], autoplay: true, interval: 3000 } },
        { type: 'paragraphNode', attrs: { color: '#FF3300', align: 'center', fontSize: 18, lineHeight: 1.6, spacing: 10 }, content: [{ type: 'text', text: '全场低至5折！错过再等一年！' }] },
        { type: 'columnsNode', attrs: { count: 2, gap: 16, items: [{ content: '' }, { content: '' }] }, content: [
          { type: 'cardNode', attrs: { bgColor: '#FFF0E5', borderRadius: 8, shadow: '0 1px 4px rgba(0,0,0,0.06)', padding: 16, title: '新品推荐' }, content: [{ type: 'paragraphNode', attrs: { color: '#333333', align: 'left', fontSize: 14, lineHeight: 1.6, spacing: 10 }, content: [{ type: 'text', text: '上新产品抢先看' }] }] },
          { type: 'cardNode', attrs: { bgColor: '#E5F5FF', borderRadius: 8, shadow: '0 1px 4px rgba(0,0,0,0.06)', padding: 16, title: '热卖榜单' }, content: [{ type: 'paragraphNode', attrs: { color: '#333333', align: 'left', fontSize: 14, lineHeight: 1.6, spacing: 10 }, content: [{ type: 'text', text: '本月热销TOP10' }] }] },
        ] },
        { type: 'buttonNode', attrs: { text: '立即抢购', url: '', bgColor: '#FF6600', textColor: '#FFFFFF', borderRadius: 24, align: 'center' } },
      ],
    },
    tags: ['促销', '折扣', '限时'],
  },
  {
    id: 'meeting-notice',
    name: '会议通知',
    description: '简洁正式的会议通知页面模板',
    category: '活动',
    coverUrl: '',
    content: {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 2, text: '会议通知', color: '#1A3C6D', align: 'center', fontSize: 28 } },
        { type: 'dividerNode', attrs: { style: 'solid', color: '#1A3C6D', width: 60, margin: 20 } },
        { type: 'cardNode', attrs: { bgColor: '#F0F4F8', borderRadius: 8, shadow: '0 1px 3px rgba(0,0,0,0.1)', padding: 20, title: '会议详情' }, content: [
          { type: 'paragraphNode', attrs: { color: '#333333', align: 'left', fontSize: 14, lineHeight: 2, spacing: 10 }, content: [{ type: 'text', text: '会议时间：2026年8月15日 14:00-16:00\n会议地点：公司总部5楼会议室\n参会人员：各部门负责人\n会议主题：Q3战略规划研讨' }] },
        ] },
        { type: 'quoteNode', attrs: { text: '请各位参会人员提前10分钟入场，并准备好相关材料。', color: '#666666', borderColor: '#1A3C6D', bgColor: '#F8FAFB', icon: '📋' } },
        { type: 'buttonNode', attrs: { text: '确认参加', url: '', bgColor: '#1A3C6D', textColor: '#FFFFFF', borderRadius: 4, align: 'center' } },
      ],
    },
    tags: ['会议', '通知', '正式'],
  },
  {
    id: 'company-news',
    name: '企业新闻',
    description: '专业大气的企业新闻页面模板',
    category: '新闻',
    coverUrl: '',
    content: {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 1, text: '公司新闻', color: '#222222', align: 'left', fontSize: 28 } },
        { type: 'imageNode', attrs: { src: '', alt: '新闻封面', width: 750, borderRadius: 4, shadow: 'none', caption: '' } },
        { type: 'paragraphNode', attrs: { color: '#444444', align: 'left', fontSize: 16, lineHeight: 1.8, spacing: 12 }, content: [{ type: 'text', text: '近日，公司在技术创新方面取得了突破性进展，成功推出了新一代智能解决方案。该产品融合了最新的人工智能技术，将为行业客户带来更高效、更智能的服务体验。' }] },
        { type: 'quoteNode', attrs: { text: '创新是驱动企业发展的核心动力。我们将持续加大研发投入，推动技术进步。', color: '#555555', borderColor: '#1890FF', bgColor: '#E6F7FF', icon: '💡' } },
        { type: 'paragraphNode', attrs: { color: '#444444', align: 'left', fontSize: 16, lineHeight: 1.8, spacing: 12 }, content: [{ type: 'text', text: '此次发布的解决方案已经在多个行业头部客户中完成了试点应用，获得了广泛好评。预计今年第四季度将正式向市场全面推广。' }] },
      ],
    },
    tags: ['企业', '新闻', '创新'],
  },
  {
    id: 'product-launch',
    name: '产品发布',
    description: '科技感十足的产品发布页面模板',
    category: '新闻',
    coverUrl: '',
    content: {
      type: 'doc',
      content: [
        { type: 'titleNode', attrs: { level: 1, text: '新品发布', color: '#7B2FBE', align: 'center', fontSize: 34 } },
        { type: 'imageNode', attrs: { src: '', alt: '产品渲染图', width: 750, borderRadius: 12, shadow: '0 4px 12px rgba(0,0,0,0.15)', caption: '全新一代旗舰产品' } },
        { type: 'dividerNode', attrs: { style: 'gradient', color: '#7B2FBE', width: 120, margin: 24 } },
        { type: 'cardNode', attrs: { bgColor: '#F5F0FA', borderRadius: 12, shadow: '0 2px 6px rgba(0,0,0,0.08)', padding: 24, title: '核心亮点' }, content: [
          { type: 'columnsNode', attrs: { count: 2, gap: 16, items: [{ content: '' }, { content: '' }] }, content: [
            { type: 'paragraphNode', attrs: { color: '#7B2FBE', align: 'center', fontSize: 15, lineHeight: 1.6, spacing: 10 }, content: [{ type: 'text', text: '性能提升200%' }] },
            { type: 'paragraphNode', attrs: { color: '#7B2FBE', align: 'center', fontSize: 15, lineHeight: 1.6, spacing: 10 }, content: [{ type: 'text', text: '续航长达48小时' }] },
          ] },
        ] },
        { type: 'buttonNode', attrs: { text: '了解更多', url: '', bgColor: '#7B2FBE', textColor: '#FFFFFF', borderRadius: 8, align: 'center' } },
      ],
    },
    tags: ['产品', '发布', '科技'],
  },
]
