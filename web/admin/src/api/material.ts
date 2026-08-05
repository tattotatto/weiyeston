import client from './client';

export interface MaterialVO {
  id: number;
  account_id: number;
  media_id?: string;
  type: string;
  name?: string;
  url: string;
  thumbnail_url?: string;
  file_size?: number;
  width?: number;
  height?: number;
  format?: string;
  created_at: string;
  updated_at: string;
}

export interface MaterialListResponse {
  list: MaterialVO[];
  total: number;
  page: number;
  page_size: number;
}

export interface MaterialListParams {
  account_id: number;
  type?: string;
  page?: number;
  size?: number;
}

// List materials (paginated, with type filter)
export function listMaterials(params: MaterialListParams) {
  return client.get<{ code: number; msg: string; data: MaterialListResponse }>('/materials', { params });
}

// Upload material (multipart form)
export function uploadMaterial(accountId: number, file: File) {
  const formData = new FormData();
  formData.append('account_id', String(accountId));
  formData.append('file', file);
  return client.post<{ code: number; msg: string; data: MaterialVO }>('/materials/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

// Get material detail
export function getMaterial(id: number) {
  return client.get<{ code: number; msg: string; data: MaterialVO }>(`/materials/${id}`);
}

// Delete material (soft delete)
export function deleteMaterial(id: number) {
  return client.delete<{ code: number; msg: string; data: null }>(`/materials/${id}`);
}
