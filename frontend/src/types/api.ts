export type SecretKind = 'text' | 'file'

export interface CreateSecretRequest {
  meta: string
  ciphertext: string
  kind: SecretKind
  ttl_seconds?: number
  views?: number
}

export interface CreateSecretResponse {
  id: string
}

export interface StatusResponse {
  max_views: number
  max_ttl_seconds: number
  max_file_bytes: number
  allowed_file_mime_types: string[]
  require_password: boolean
}

export interface ConsumeSecretResponse {
  meta: string
  ciphertext: string
  kind: SecretKind
}
