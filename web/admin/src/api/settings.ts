import client from './client';

export interface StorageSettingsResponse {
  driver: string;
  local_path: string;
  s3_endpoint: string;
  s3_bucket: string;
  s3_region: string;
  s3_key: string; // 脱敏显示
  public_url: string;
}

export interface StorageSettingsRequest {
  driver: string;
  local_path: string;
  s3_endpoint: string;
  s3_bucket: string;
  s3_region: string;
  s3_key: string;
  s3_secret: string;
  public_url: string;
}

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// 获取存储配置
export function getStorageSettings(): Promise<{ data: ApiResponse<StorageSettingsResponse> }> {
  return client.get('/admin/settings');
}

// 更新存储配置
export function updateStorageSettings(
  data: StorageSettingsRequest,
): Promise<{ data: ApiResponse<null> }> {
  return client.put('/admin/settings', data);
}
