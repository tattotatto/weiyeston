import client from './client';

export interface CreateAccountParams {
  name: string;
  wx_app_id: string;
  wx_app_secret: string;
  wx_original_id?: string;
  description?: string;
  avatar_url?: string;
  qr_code_url?: string;
}

export interface UpdateAccountParams {
  name?: string;
  wx_app_id?: string;
  wx_app_secret?: string;
  wx_original_id?: string;
  description?: string;
  avatar_url?: string;
  qr_code_url?: string;
}

export interface AccountListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  auth_type?: number;
  auth_status?: number;
}

export interface AccountVO {
  id: number;
  tenant_id: number;
  name: string;
  wx_app_id: string;
  wx_original_id?: string;
  auth_type: number;
  auth_status: number;
  description?: string;
  avatar_url?: string;
  qr_code_url?: string;
  fans_count: number;
  nick_name?: string;
  head_img?: string;
  principal_name?: string;
  token_expire_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// Create account (manual setup)
export function createAccount(data: CreateAccountParams): Promise<{ data: ApiResponse<AccountVO> }> {
  return client.post('/accounts', data);
}

// List accounts (paginated)
export function listAccounts(params: AccountListParams): Promise<{ data: ApiResponse<PaginatedResponse<AccountVO>> }> {
  return client.get('/accounts', { params });
}

// Get account detail
export function getAccount(id: number): Promise<{ data: ApiResponse<AccountVO> }> {
  return client.get(`/accounts/${id}`);
}

// Update account
export function updateAccount(id: number, data: UpdateAccountParams): Promise<{ data: ApiResponse<AccountVO> }> {
  return client.put(`/accounts/${id}`, data);
}

// Delete account (soft delete)
export function deleteAccount(id: number): Promise<{ data: ApiResponse<null> }> {
  return client.delete(`/accounts/${id}`);
}
