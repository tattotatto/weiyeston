import client from './client';

export interface NewsVO {
  id: number;
  account_id: number;
  channel_id: number;
  author_name?: string;
  author_avatar?: string;
  content: string;
  like_count: number;
  comment_count: number;
  status: number;
  status_text: string;
  is_top: boolean;
  created_at: string;
  updated_at: string;
}

export interface NewsUserVO {
  id: number;
  account_id: number;
  openid: string;
  nickname?: string;
  avatar_url?: string;
  status: number;
  created_at: string;
}

export interface CreateNewsParams {
  channel_id: number;
  content: string;
  author_name?: string;
  status?: number;
  is_top?: boolean;
}

export interface UpdateNewsParams {
  channel_id?: number;
  content?: string;
  author_name?: string;
  status?: number;
  is_top?: boolean;
}

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// News CRUD

export function listNews(params?: { page?: number; size?: number }): Promise<{ data: ApiResponse<{ list: NewsVO[]; page: number; page_size: number }> }> {
  return client.get('/news', { params });
}

export function getNews(id: number): Promise<{ data: ApiResponse<NewsVO> }> {
  return client.get(`/news/${id}`);
}

export function createNews(data: CreateNewsParams): Promise<{ data: ApiResponse<NewsVO> }> {
  return client.post('/news', data);
}

export function updateNews(id: number, data: UpdateNewsParams): Promise<{ data: ApiResponse<null> }> {
  return client.put(`/news/${id}`, data);
}

export function deleteNews(id: number): Promise<{ data: ApiResponse<null> }> {
  return client.delete(`/news/${id}`);
}

// Users

export function listNewsUsers(params?: { account_id?: number; page?: number; size?: number }): Promise<{ data: ApiResponse<{ list: NewsUserVO[]; page: number; page_size: number }> }> {
  return client.get('/quicknews/users', { params });
}
