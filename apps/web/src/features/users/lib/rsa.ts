// ============================================================
// RSA Encryption Utility — User Management
// ============================================================
// Fetches the RSA public key from the backend and encrypts
// passwords using the Web Crypto API (RSA-OAEP + SHA-256).
// The public key is cached per session to avoid repeated fetches.
// ============================================================

import { apiClient } from '@/lib/api-client';

type PublicKeyResponse = {
  publicKey: string;
};

/** Module-level cache for the public key PEM string */
let cachedPublicKey: string | null = null;

/**
 * Fetch the RSA public key from the backend.
 * Caches the result so subsequent calls within the same session
 * return immediately without a network request.
 */
export async function fetchPublicKey(): Promise<string> {
  if (cachedPublicKey) {
    return cachedPublicKey;
  }

  const res = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
  const key = res.publicKey;
  cachedPublicKey = key;
  return key;
}

/**
 * Parse a PEM-encoded public key into raw DER bytes.
 * Strips the PEM header/footer and decodes the base64 body.
 */
function pemToArrayBuffer(pem: string): ArrayBuffer {
  const lines = pem
    .replace(/-----BEGIN PUBLIC KEY-----/, '')
    .replace(/-----END PUBLIC KEY-----/, '')
    .replace(/\s/g, '');
  const binary = atob(lines);
  const buffer = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    buffer[i] = binary.charCodeAt(i);
  }
  return buffer.buffer;
}

/**
 * Encrypt a password using RSA-OAEP with SHA-256.
 *
 * 1. Fetches (or retrieves cached) PEM public key from the API
 * 2. Parses the PEM to extract raw key bytes
 * 3. Imports using window.crypto.subtle.importKey with RSA-OAEP + SHA-256
 * 4. Encrypts the password
 * 5. Returns base64-encoded ciphertext
 *
 * Compatible with the Go backend's rsa.DecryptOAEP(sha256.New(), ...).
 */
export async function encryptPassword(password: string): Promise<string> {
  const publicKeyPem = await fetchPublicKey();
  const keyData = pemToArrayBuffer(publicKeyPem);

  const cryptoKey = await window.crypto.subtle.importKey(
    'spki',
    keyData,
    {
      name: 'RSA-OAEP',
      hash: 'SHA-256'
    },
    false,
    ['encrypt']
  );

  const encoded = new TextEncoder().encode(password);
  const encrypted = await window.crypto.subtle.encrypt({ name: 'RSA-OAEP' }, cryptoKey, encoded);

  // Convert ArrayBuffer to base64
  const bytes = new Uint8Array(encrypted);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}
