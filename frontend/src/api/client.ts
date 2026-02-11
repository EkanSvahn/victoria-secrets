import type { ConsumeSecretResponse, CreateSecretRequest, CreateSecretResponse, StatusResponse } from '../types/api'

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

type ErrorPayload = {
  error?: string
  message?: string
}

export class ApiRequestError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = code
  }
}

function parseErrorBody(raw: string): ErrorPayload {
  if (!raw) return {}
  try {
    return JSON.parse(raw) as ErrorPayload
  } catch {
    return {}
  }
}

async function parseJson<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const body = await response.text()
    const parsed = parseErrorBody(body)
    const message = parsed.message?.trim() || `Request failed (${response.status})`
    const code = parsed.error?.trim() || 'request_failed'
    throw new ApiRequestError(response.status, code, message)
  }
  return (await response.json()) as T
}

export async function createSecret(payload: CreateSecretRequest): Promise<CreateSecretResponse> {
  const response = await fetch(`${BASE_URL}/v1/secrets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })
  return parseJson<CreateSecretResponse>(response)
}

export async function getStatus(): Promise<StatusResponse> {
  const response = await fetch(`${BASE_URL}/status`)
  return parseJson<StatusResponse>(response)
}

export async function consumeSecret(id: string): Promise<ConsumeSecretResponse> {
  const response = await fetch(`${BASE_URL}/v1/secrets/${encodeURIComponent(id)}/consume`, {
    method: 'POST'
  })
  return parseJson<ConsumeSecretResponse>(response)
}

export async function previewSecret(id: string): Promise<boolean> {
  const response = await fetch(`${BASE_URL}/v1/secrets/${encodeURIComponent(id)}`)
  return response.ok
}

export function toApiErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiRequestError) {
    if (error.code === 'not_found') {
      return 'Secret not found. It may already be consumed or expired.'
    }
    if (error.code === 'rate_limited' || error.status === 429) {
      return 'Too many requests. Wait a moment and try again.'
    }
    if (error.code === 'invalid_input') {
      return error.message
    }
    if (error.code === 'internal_error' || error.status >= 500) {
      return 'Server error while processing request. Please try again.'
    }
    return error.message || fallback
  }
  if (error instanceof TypeError) {
    return 'Cannot reach API server. Check that backend is running and API URL is correct.'
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  return fallback
}
