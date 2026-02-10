function toBase64Url(bytes: Uint8Array): string {
  const binary = String.fromCharCode(...bytes)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function fromBase64Url(input: string): Uint8Array {
  const base64 = input.replace(/-/g, '+').replace(/_/g, '/') + '==='.slice((input.length + 3) % 4)
  const binary = atob(base64)
  return Uint8Array.from(binary, (c) => c.charCodeAt(0))
}

function encodeUtf8(value: string): Uint8Array {
  return new TextEncoder().encode(value)
}

function decodeUtf8(value: Uint8Array): string {
  return new TextDecoder().decode(value)
}

async function deriveKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
  const material = await crypto.subtle.importKey('raw', encodeUtf8(password), 'PBKDF2', false, ['deriveKey'])
  return crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      hash: 'SHA-256',
      salt,
      iterations: 310000
    },
    material,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  )
}

async function importAesKey(raw: Uint8Array): Promise<CryptoKey> {
  return crypto.subtle.importKey('raw', raw, { name: 'AES-GCM' }, false, ['encrypt', 'decrypt'])
}

async function exportAesKey(key: CryptoKey): Promise<Uint8Array> {
  const raw = await crypto.subtle.exportKey('raw', key)
  return new Uint8Array(raw)
}

export interface EncryptResult {
  ciphertext: string
  meta: string
  linkSecret: string
}

export async function encryptText(plaintext: string, password?: string): Promise<EncryptResult> {
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const salt = crypto.getRandomValues(new Uint8Array(16))

  let key: CryptoKey
  let meta: Record<string, string | number> = { v: 1, t: 'text', alg: 'AES-GCM-256' }
  let linkSecret = ''

  if (password && password.trim() !== '') {
    key = await deriveKey(password, salt)
    meta = { ...meta, kdf: 'PBKDF2-SHA256', i: 310000, s: toBase64Url(salt) }
  } else {
    key = await crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, ['encrypt', 'decrypt'])
    linkSecret = toBase64Url(await exportAesKey(key))
  }

  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, encodeUtf8(plaintext))
  const payload = {
    iv: toBase64Url(iv),
    ct: toBase64Url(new Uint8Array(encrypted))
  }

  return {
    ciphertext: JSON.stringify(payload),
    meta: JSON.stringify(meta),
    linkSecret
  }
}

export async function decryptText(ciphertext: string, metaRaw: string, linkSecret: string, password?: string): Promise<string> {
  const payload = JSON.parse(ciphertext) as { iv: string; ct: string }
  const meta = JSON.parse(metaRaw) as { s?: string }

  let key: CryptoKey
  if (password && meta.s) {
    key = await deriveKey(password, fromBase64Url(meta.s))
  } else {
    key = await importAesKey(fromBase64Url(linkSecret))
  }

  const plaintext = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: fromBase64Url(payload.iv) },
    key,
    fromBase64Url(payload.ct)
  )
  return decodeUtf8(new Uint8Array(plaintext))
}
