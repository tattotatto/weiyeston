import client from './client';

export interface ChannelVO {
  id: number;
  account_id: number;
  parent_id?: number;
  name: string;
  slug?: string;
  level: number;
  sort_order: number;
  cover_url?: string;
  description?: string;
  status: number;
  children?: ChannelVO[];
  created_at: string;
  updated_at: string;
}

export interface ArticleVO {
  id: number;
  account_id: number;
  channel_id?: number;
  title?: string;
  cover_url?: string;
  summary?: string;
  author?: string;
  content: any;
  status: number;
  status_text: string;
  is_template: boolean;
  template_cat?: string;
  sort_order: number;
  view_count: number;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateChannelParams {
  name: string;
  parent_id?: number;
  slug?: string;
  level?: number;
  sort_order?: number;
  cover_url?: string;
  description?: string;
  status?: number;
}

export interface UpdateChannelParams {
  name?: string;
  parent_id?: number;
  slug?: string;
  level?: number;
  sort_order?: number;
  cover_url?: string;
  description?: string;
  status?: number;
}

export interface CreateArticleParams {
  channel_id?: number;
  title?: string;
  cover_url?: string;
  summary?: string;
  author?: string;
  content: any;
  status?: number;
  sort_order?: number;
  is_template?: boolean;
  template_cat?: string;
}

export interface UpdateArticleParams {
  channel_id?: number;
  title?: string;
  cover_url?: string;
  summary?: string;
  author?: string;
  content?: any;
  status?: number;
  sort_order?: number;
  is_template?: boolean;
  template_cat?: string;
}

export interface ArticleListParams {
  channel_id?: number;
  status?: number;
  page?: number;
  size?: number;
}

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// Channels

export function listChannels(): Promise<{ data: ApiResponse<ChannelVO[]> }> {
  return client.get('/cms/channels');
}

export function createChannel(data: CreateChannelParams): Promise<{ data: ApiResponse<ChannelVO> }> {
  return client.post('/cms/channels', data);
}

export function updateChannel(id: number, data: UpdateChannelParams): Promise<{ data: ApiResponse<null> }> {
  return client.put(`/cms/channels/${id}`, data);
}

export function deleteChannel(id: number): Promise<{ data: ApiResponse<null> }> {
  return client.delete(`/cms/channels/${id}`);
}

// Articles

export function listArticles(params?: ArticleListParams): Promise<{ data: ApiResponse<{ list: ArticleVO[]; page: number; page_size: number }> }> {
  return client.get('/cms/articles', { params });
}

export function getArticle(id: number): Promise<{ data: ApiResponse<ArticleVO> }> {
  return client.get(`/cms/articles/${id}`);
}

export function createArticle(data: CreateArticleParams): Promise<{ data: ApiResponse<ArticleVO> }> {
  return client.post('/cms/articles', data);
}

export function updateArticle(id: number, data: UpdateArticleParams): Promise<{ data: ApiResponse<null> }> {
  return client.put(`/cms/articles/${id}`, data);
}

export function deleteArticle(id: number): Promise<{ data: ApiResponse<null> }> {
  return client.delete(`/cms/articles/${id}`);
}

export function previewArticle(id: number): Promise<{ data: ApiResponse<ArticleVO> }> {
  return client.get(`/cms/articles/${id}/preview`);
}
