import client from './client'
import { message } from 'antd'

export interface AIWriteRequest {
  title: string
  keywords?: string
  style?: string
  max_words?: number
}

export interface AIWriteResponse {
  content: string
  usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
}

export interface AILayoutRequest {
  content: string // JSON string
}

export interface AILayoutResponse {
  content: Record<string, unknown>
  usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
}

export interface AIProofreadRequest {
  text: string
}

export interface Correction {
  position: number
  length: number
  type: 'typo' | 'grammar' | 'sensitive' | 'style'
  original: string
  suggestion: string
  explanation: string
}

export interface AIProofreadResponse {
  corrections: Correction[]
  usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
}

export async function aiWrite(data: AIWriteRequest): Promise<AIWriteResponse> {
  try {
    const res = await client.post('/ai/write', data)
    return res.data.data
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { msg?: string } } })?.response?.data?.msg || 'AI写作请求失败'
    message.error(msg)
    throw err
  }
}

export async function aiLayout(data: AILayoutRequest): Promise<AILayoutResponse> {
  try {
    const res = await client.post('/ai/layout', data)
    return res.data.data
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { msg?: string } } })?.response?.data?.msg || 'AI排版请求失败'
    message.error(msg)
    throw err
  }
}

export async function aiProofread(data: AIProofreadRequest): Promise<AIProofreadResponse> {
  try {
    const res = await client.post('/ai/proofread', data)
    return res.data.data
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { msg?: string } } })?.response?.data?.msg || 'AI校对请求失败'
    message.error(msg)
    throw err
  }
}
