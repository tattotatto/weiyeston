import client from './client';

export interface MenuVO {
  id: number;
  account_id: number;
  menu_json: unknown;
  status: number; // 0=草稿 1=已发布
  status_text: string;
  published_at?: string;
  updated_at: string;
}

export interface SaveMenuParams {
  menu_json: unknown;
}

// Get current menu
export function getMenu(accountId: number) {
  return client.get<{ code: number; msg: string; data: MenuVO | null }>(`/accounts/${accountId}/menu`);
}

// Save draft
export function saveMenuDraft(accountId: number, data: SaveMenuParams) {
  return client.post<{ code: number; msg: string; data: MenuVO }>(`/accounts/${accountId}/menu`, data);
}

// Publish menu to WeChat
export function publishMenu(accountId: number) {
  return client.put<{ code: number; msg: string; data: MenuVO }>(`/accounts/${accountId}/menu/publish`);
}

// Delete draft
export function deleteMenuDraft(accountId: number) {
  return client.delete<{ code: number; msg: string; data: null }>(`/accounts/${accountId}/menu`);
}
