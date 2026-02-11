import type { ConsumeSecretResponse, CreateSecretRequest, CreateSecretResponse } from '../types/api'

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

async function parseJson<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const body = await response.text()
    throw new Error(`request failed (${response.status}): ${body}`)
  }
  return (await response.json()) as T
}

export async function createSecret(payload: CreateSecretRequest): Promise<CreateSecretResponse> {
  const response = await fetch(`${BASE_URL}/api/v1/secrets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })
  return parseJson<CreateSecretResponse>(response)
}

export async function consumeSecret(id: string): Promise<ConsumeSecretResponse> {
  const response = await fetch(`${BASE_URL}/api/v1/secrets/${encodeURIComponent(id)}/consume`, {
    method: 'POST'
  })
  return parseJson<ConsumeSecretResponse>(response)
}

export async function previewSecret(id: string): Promise<boolean> {
  const response = await fetch(`${BASE_URL}/api/v1/secrets/${encodeURIComponent(id)}`)
  return response.ok
}
