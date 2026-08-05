import client from './client';

export interface ReplyVO {
  id: number;
  account_id: number;
  keyword?: string;
  match_type: number; // 0=精确 1=模糊
  reply_type: number; // 1=文本 2=图文
  reply_content: string;
  reply_title?: string;
  reply_desc?: string;
  reply_cover_url?: string;
  reply_url?: string;
  status: number; // 0=停用 1=启用
  sort_order: number;
}

export interface CreateReplyParams {
  keyword?: string;
  match_type: number;
  reply_type: number;
  reply_content: string;
  reply_title?: string;
  reply_desc?: string;
  reply_cover_url?: string;
  reply_url?: string;
  status?: number;
  sort_order?: number;
}

export interface UpdateReplyParams {
  keyword?: string;
  match_type?: number;
  reply_type?: number;
  reply_content?: string;
  reply_title?: string;
  reply_desc?: string;
  reply_cover_url?: string;
  reply_url?: string;
  status?: number;
  sort_order?: number;
}

// List reply rules by account
export function listReplies(accountId: number) {
  return client.get<{ code: number; msg: string; data: ReplyVO[] }>(`/accounts/${accountId}/replies`);
}

// Create a reply rule
export function createReply(accountId: number, data: CreateReplyParams) {
  return client.post<{ code: number; msg: string; data: ReplyVO }>(`/accounts/${accountId}/replies`, data);
}

// Update a reply rule
export function updateReply(ruleId: number, data: UpdateReplyParams) {
  return client.put<{ code: number; msg: string; data: ReplyVO }>(`/replies/${ruleId}`, data);
}

// Delete a reply rule
export function deleteReply(ruleId: number) {
  return client.delete<{ code: number; msg: string; data: null }>(`/replies/${ruleId}`);
}
