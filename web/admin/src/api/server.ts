import client from './client';

export interface ServerInfo {
  public_ip: string;
}

export interface ServerInfoResponse {
  code: number;
  msg: string;
  data: ServerInfo;
}

// 获取服务器信息（包含公网 IP）
export function getServerInfo(): Promise<{ data: ServerInfoResponse }> {
  return client.get('/server/info');
}
