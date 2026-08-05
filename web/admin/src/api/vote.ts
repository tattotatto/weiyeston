import client from './client';

export interface VoteOptionVO {
  id: number;
  vote_id: number;
  content: string;
  image_url?: string;
  sort_order: number;
  vote_count: number;
}

export interface VoteVO {
  id: number;
  account_id: number;
  title: string;
  description?: string;
  cover_url?: string;
  vote_type: number;
  max_choices: number;
  max_votes: number;
  start_time?: string;
  end_time?: string;
  total_votes: number;
  status: number;
  status_text: string;
  options?: VoteOptionVO[];
  created_at: string;
  updated_at: string;
}

export interface VoteResultVO {
  vote_id: number;
  title: string;
  total_votes: number;
  options: VoteOptionVO[];
}

export interface CreateOptionItem {
  content: string;
  image_url?: string;
  sort_order?: number;
}

export interface CreateVoteParams {
  title: string;
  description?: string;
  cover_url?: string;
  vote_type?: number;
  max_choices?: number;
  max_votes?: number;
  start_time?: string;
  end_time?: string;
  status?: number;
  options: CreateOptionItem[];
}

export interface UpdateVoteParams {
  title?: string;
  description?: string;
  cover_url?: string;
  vote_type?: number;
  max_choices?: number;
  max_votes?: number;
  start_time?: string;
  end_time?: string;
  status?: number;
  options?: CreateOptionItem[];
}

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// Votes CRUD

export function listVotes(params?: { page?: number; size?: number }): Promise<{ data: ApiResponse<{ list: VoteVO[]; page: number; page_size: number }> }> {
  return client.get('/votes', { params });
}

export function getVote(id: number): Promise<{ data: ApiResponse<VoteVO> }> {
  return client.get(`/votes/${id}`);
}

export function createVote(data: CreateVoteParams): Promise<{ data: ApiResponse<VoteVO> }> {
  return client.post('/votes', data);
}

export function updateVote(id: number, data: UpdateVoteParams): Promise<{ data: ApiResponse<null> }> {
  return client.put(`/votes/${id}`, data);
}

export function deleteVote(id: number): Promise<{ data: ApiResponse<null> }> {
  return client.delete(`/votes/${id}`);
}

// Vote Results

export function getVoteResults(id: number): Promise<{ data: ApiResponse<VoteResultVO> }> {
  return client.get(`/votes/${id}/results`);
}
