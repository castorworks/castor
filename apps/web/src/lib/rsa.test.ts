import forge from 'node-forge';
import { describe, expect, it } from 'vitest';
import { encryptPassword } from './rsa';

describe('encryptPassword', () => {
  it('encrypts with RSA-OAEP using SHA-256 for Go backend compatibility', () => {
    const keyPair = forge.pki.rsa.generateKeyPair({ bits: 2048 });
    const publicKeyPem = forge.pki.publicKeyToPem(keyPair.publicKey);
    const plaintext = 'password123';

    const encrypted = encryptPassword(publicKeyPem, plaintext);
    const decrypted = keyPair.privateKey.decrypt(forge.util.decode64(encrypted), 'RSA-OAEP', {
      md: forge.md.sha256.create(),
      mgf1: {
        md: forge.md.sha256.create()
      }
    });

    expect(encrypted).not.toBe(plaintext);
    expect(decrypted).toBe(plaintext);
  });
});
