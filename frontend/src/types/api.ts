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

export interface ConsumeSecretResponse {
  meta: string
  ciphertext: string
  kind: SecretKind
}
