import forge from 'node-forge';

/**
 * Encrypt a password using RSA-OAEP with SHA-256.
 *
 * The backend decrypts with Go's rsa.DecryptOAEP(sha256.New(), ...), so the
 * frontend must use SHA-256 for both the OAEP digest and MGF1 digest.
 */
export function encryptPassword(publicKeyPem: string, plaintext: string): string {
  const publicKey = forge.pki.publicKeyFromPem(publicKeyPem);
  const encrypted = publicKey.encrypt(plaintext, 'RSA-OAEP', {
    md: forge.md.sha256.create(),
    mgf1: {
      md: forge.md.sha256.create()
    }
  });

  return forge.util.encode64(encrypted);
}
