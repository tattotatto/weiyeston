import client from './client';

interface LoginParams {
  username: string;
  password: string;
}

interface LoginData {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: number;
    username: string;
    nickname: string;
    role: string;
    avatar_url?: string;
  };
}

interface LoginResponse {
  code: number;
  msg: string;
  data: LoginData;
}

interface RefreshData {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface RefreshResponse {
  code: number;
  msg: string;
  data: RefreshData;
}

interface UserInfo {
  id: number;
  username: string;
  nickname: string;
  role: string;
  avatar_url?: string;
}

interface MeResponse {
  code: number;
  msg: string;
  data: UserInfo;
}

interface LogoutResponse {
  code: number;
  msg: string;
}

// 登录
export async function login(params: LoginParams): Promise<LoginResponse> {
  const { data } = await client.post<LoginResponse>('/auth/login', params);
  return data;
}

// 刷新 Token
export async function refreshToken(refresh_token: string): Promise<RefreshResponse> {
  const { data } = await client.post<RefreshResponse>('/auth/refresh', { refresh_token });
  return data;
}

// 获取当前用户信息
export async function getMe(): Promise<MeResponse> {
  const { data } = await client.get<MeResponse>('/auth/me');
  return data;
}

// 登出
export async function logout(): Promise<LogoutResponse> {
  const { data } = await client.post<LogoutResponse>('/auth/logout');
  return data;
}

// 注册
export interface RegisterParams {
  username: string;
  password: string;
  email: string;
  nickname?: string;
}

export interface RegisterData {
  id: number;
  username: string;
}

export interface RegisterResponse {
  code: number;
  msg: string;
  data: RegisterData;
}

export async function register(params: RegisterParams): Promise<RegisterResponse> {
  const { data } = await client.post<RegisterResponse>('/auth/register', params);
  return data;
}
