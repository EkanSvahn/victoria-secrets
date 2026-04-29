import { describe, expect, it } from 'vitest'
import { decryptFile, decryptText, encryptFile, encryptText } from './crypto'

function toBase64Url(bytes: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000))
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

async function craftLegacyPBKDF2Secret(plaintext: string, password: string, iterations = 310000): Promise<{ ciphertext: string; meta: string }> {
  const salt = crypto.getRandomValues(new Uint8Array(16))
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const material = await crypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveKey'])
  const key = await crypto.subtle.deriveKey(
    { name: 'PBKDF2', hash: 'SHA-256', salt, iterations },
    material,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  )
  const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, new TextEncoder().encode(plaintext))
  const ciphertext = JSON.stringify({ iv: toBase64Url(iv), ct: toBase64Url(new Uint8Array(ct)) })
  const meta = JSON.stringify({
    v: 1,
    t: 'text',
    alg: 'AES-GCM-256',
    kdf: 'PBKDF2-SHA256',
    i: iterations,
    s: toBase64Url(salt)
  })
  return { ciphertext, meta }
}

describe('encryptText / decryptText round-trip', () => {
  it('round-trips with link-fragment key (no password)', async () => {
    const plaintext = 'hej världen — secret note 🔑'
    const result = await encryptText(plaintext)

    expect(result.linkSecret.length).toBeGreaterThan(0)
    expect(result.meta).toMatch(/"alg":"AES-GCM-256"/)
    expect(JSON.parse(result.meta).kdf).toBeUndefined()

    const decrypted = await decryptText(result.ciphertext, result.meta, result.linkSecret)
    expect(decrypted).toBe(plaintext)
  })

  it('round-trips with Argon2id-derived password key', async () => {
    const plaintext = 'rotated db password'
    const password = 'correct horse battery staple'
    const result = await encryptText(plaintext, password)

    expect(result.linkSecret).toBe('')
    const meta = JSON.parse(result.meta)
    expect(meta.kdf).toBe('ARGON2ID')
    expect(meta.tt).toBe(3)
    expect(meta.tm).toBe(65536)
    expect(meta.tp).toBe(1)
    expect(typeof meta.s).toBe('string')

    const decrypted = await decryptText(result.ciphertext, result.meta, '', password)
    expect(decrypted).toBe(plaintext)
  })

  it('rejects decryption with wrong password', async () => {
    const result = await encryptText('top secret', 'right-password-12345')
    await expect(decryptText(result.ciphertext, result.meta, '', 'wrong-password')).rejects.toBeDefined()
  })

  it('rejects tampered ciphertext (AES-GCM auth tag)', async () => {
    const result = await encryptText('payload')
    const payload = JSON.parse(result.ciphertext) as { iv: string; ct: string }
    const tampered = JSON.stringify({ ...payload, ct: payload.ct.slice(0, -2) + (payload.ct.endsWith('A') ? 'B' : 'A') })
    await expect(decryptText(tampered, result.meta, result.linkSecret)).rejects.toBeDefined()
  })

  it('handles plaintext of varying lengths (covers base64url padding cases)', async () => {
    for (const length of [0, 1, 2, 3, 4, 5, 17, 4096]) {
      const plaintext = 'x'.repeat(length)
      const result = await encryptText(plaintext)
      const decrypted = await decryptText(result.ciphertext, result.meta, result.linkSecret)
      expect(decrypted).toBe(plaintext)
    }
  })
})

describe('encryptFile / decryptFile round-trip', () => {
  function makeFile(name: string, type: string, contents: Uint8Array): File {
    return new File([contents as unknown as BlobPart], name, { type })
  }

  it('round-trips file content, name, and mime with link-fragment key', async () => {
    const bytes = new Uint8Array([0x25, 0x50, 0x44, 0x46, 0x2d, 0x31, 0x2e, 0x34]) // %PDF-1.4
    const file = makeFile('report.pdf', 'application/pdf', bytes)
    const encrypted = await encryptFile(file)

    const meta = JSON.parse(encrypted.meta)
    expect(meta.t).toBe('file')
    expect(meta.n).toBe('report.pdf')
    expect(meta.m).toBe('application/pdf')
    expect(meta.z).toBe(bytes.length)

    const decrypted = await decryptFile(encrypted.ciphertext, encrypted.meta, encrypted.linkSecret)
    expect(decrypted.name).toBe('report.pdf')
    expect(decrypted.type).toBe('application/pdf')
    const restored = new Uint8Array(await decrypted.arrayBuffer())
    expect(Array.from(restored)).toEqual(Array.from(bytes))
  })

  it('round-trips file with Argon2id password', async () => {
    const bytes = crypto.getRandomValues(new Uint8Array(2048))
    const file = makeFile('blob.bin', 'application/octet-stream', bytes)
    const encrypted = await encryptFile(file, 'super-secret-passphrase')

    const decrypted = await decryptFile(encrypted.ciphertext, encrypted.meta, '', 'super-secret-passphrase')
    const restored = new Uint8Array(await decrypted.arrayBuffer())
    expect(Array.from(restored)).toEqual(Array.from(bytes))
  })
})

describe('PBKDF2 legacy backward-compat', () => {
  it('decrypts secrets created with PBKDF2-SHA256 metadata', async () => {
    const plaintext = 'created before Argon2id rollout'
    const password = 'legacy-password'
    const legacy = await craftLegacyPBKDF2Secret(plaintext, password)

    const decrypted = await decryptText(legacy.ciphertext, legacy.meta, '', password)
    expect(decrypted).toBe(plaintext)
  })

  it('decrypts PBKDF2 secret when meta omits kdf field (older format)', async () => {
    const plaintext = 'oldest format'
    const password = 'legacy'
    const legacy = await craftLegacyPBKDF2Secret(plaintext, password)
    const meta = JSON.parse(legacy.meta)
    delete meta.kdf
    const decrypted = await decryptText(legacy.ciphertext, JSON.stringify(meta), '', password)
    expect(decrypted).toBe(plaintext)
  })
})

describe('cryptographic uniqueness', () => {
  it('generates unique IV and link secret across many no-password encryptions', async () => {
    const ivs = new Set<string>()
    const linkSecrets = new Set<string>()
    const iterations = 64

    for (let i = 0; i < iterations; i++) {
      const result = await encryptText('same plaintext')
      ivs.add(JSON.parse(result.ciphertext).iv as string)
      linkSecrets.add(result.linkSecret)
    }

    expect(ivs.size).toBe(iterations)
    expect(linkSecrets.size).toBe(iterations)
  })

  it('generates unique salt and IV across password encryptions', async () => {
    const ivs = new Set<string>()
    const salts = new Set<string>()
    const iterations = 4

    for (let i = 0; i < iterations; i++) {
      const result = await encryptText('same plaintext', 'same-password')
      ivs.add(JSON.parse(result.ciphertext).iv as string)
      salts.add(JSON.parse(result.meta).s as string)
    }

    expect(ivs.size).toBe(iterations)
    expect(salts.size).toBe(iterations)
  })

  it('encrypting same plaintext twice produces different ciphertext', async () => {
    const a = await encryptText('repeat')
    const b = await encryptText('repeat')
    expect(a.ciphertext).not.toBe(b.ciphertext)
    expect(a.linkSecret).not.toBe(b.linkSecret)
  })
})
