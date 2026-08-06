import client from './client';

export interface AdminUserVO {
  id: number;
  username: string;
  nickname: string;
  role: string;
  status: number; // 0=待审核, 1=已通过, 2=已禁用
  vip_level: string;
  vip_end_time: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface AdminUserListResponse {
  code: number;
  msg: string;
  data: AdminUserVO[];
}

export interface AdminUserUpdateParams {
  status?: number;
  vip_level?: string;
  vip_end_time?: string;
}

export interface AdminUserUpdateResponse {
  code: number;
  msg: string;
  data: AdminUserVO;
}

// 获取所有租户列表（超级管理员）
export function getUsers(): Promise<{ data: AdminUserListResponse }> {
  return client.get('/admin/users');
}

// 更新用户信息（超级管理员）
export function updateUser(id: number, data: AdminUserUpdateParams): Promise<{ data: AdminUserUpdateResponse }> {
  return client.put(`/admin/users/${id}`, data);
}
