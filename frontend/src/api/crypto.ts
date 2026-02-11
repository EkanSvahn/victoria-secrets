function bytesToBinary(bytes: Uint8Array): string {
  let binary = ''
  const chunk = 0x8000
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk))
  }
  return binary
}

function toBase64Url(bytes: Uint8Array): string {
  const binary = bytesToBinary(bytes)
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

function randomIV(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(12))
}

function randomSalt(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(16))
}

interface SecretMeta {
  v: 1
  t: 'text' | 'file'
  alg: 'AES-GCM-256'
  kdf?: 'PBKDF2-SHA256'
  i?: 310000
  s?: string
  n?: string
  m?: string
  z?: number
}

interface Payload {
  iv: string
  ct: string
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

async function resolveKey(password?: string): Promise<{ key: CryptoKey; linkSecret: string; metaPatch: Partial<SecretMeta> }> {
  const salt = randomSalt()
  let key: CryptoKey
  let linkSecret = ''
  let metaPatch: Partial<SecretMeta> = {}

  if (password && password.trim() !== '') {
    key = await deriveKey(password, salt)
    metaPatch = { kdf: 'PBKDF2-SHA256', i: 310000, s: toBase64Url(salt) }
  } else {
    key = await crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, ['encrypt', 'decrypt'])
    linkSecret = toBase64Url(await exportAesKey(key))
  }
  return { key, linkSecret, metaPatch }
}

export async function encryptText(plaintext: string, password?: string): Promise<EncryptResult> {
  const iv = randomIV()
  const { key, linkSecret, metaPatch } = await resolveKey(password)
  const meta: SecretMeta = { v: 1, t: 'text', alg: 'AES-GCM-256', ...metaPatch }

  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, encodeUtf8(plaintext))
  const payload: Payload = {
    iv: toBase64Url(iv),
    ct: toBase64Url(new Uint8Array(encrypted))
  }

  return {
    ciphertext: JSON.stringify(payload),
    meta: JSON.stringify(meta),
    linkSecret
  }
}

async function decryptBytes(ciphertext: string, metaRaw: string, linkSecret: string, password?: string): Promise<Uint8Array> {
  const payload = JSON.parse(ciphertext) as Payload
  const meta = JSON.parse(metaRaw) as SecretMeta

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
  return new Uint8Array(plaintext)
}

export async function decryptText(ciphertext: string, metaRaw: string, linkSecret: string, password?: string): Promise<string> {
  const plaintext = await decryptBytes(ciphertext, metaRaw, linkSecret, password)
  return decodeUtf8(plaintext)
}

export async function encryptFile(file: File, password?: string): Promise<EncryptResult> {
  const iv = randomIV()
  const { key, linkSecret, metaPatch } = await resolveKey(password)
  const raw = new Uint8Array(await file.arrayBuffer())
  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, raw)

  const payload: Payload = {
    iv: toBase64Url(iv),
    ct: toBase64Url(new Uint8Array(encrypted))
  }
  const meta: SecretMeta = {
    v: 1,
    t: 'file',
    alg: 'AES-GCM-256',
    n: file.name || 'secret.bin',
    m: file.type || 'application/octet-stream',
    z: file.size,
    ...metaPatch
  }

  return {
    ciphertext: JSON.stringify(payload),
    meta: JSON.stringify(meta),
    linkSecret
  }
}

export async function decryptFile(ciphertext: string, metaRaw: string, linkSecret: string, password?: string): Promise<File> {
  const meta = JSON.parse(metaRaw) as SecretMeta
  const plaintext = await decryptBytes(ciphertext, metaRaw, linkSecret, password)
  const name = meta.n || 'secret.bin'
  const type = meta.m || 'application/octet-stream'
  return new File([plaintext], name, { type })
}
