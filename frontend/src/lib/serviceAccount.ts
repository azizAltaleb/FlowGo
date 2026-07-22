export interface GeneratedServiceAccountKeyPair {
  publicKeyPem: string;
  privateKeyPem: string;
}

export interface ServiceAccountProfile {
  type: "serviceaccount";
  keyId: string;
  key: string;
  userId: string;
  issuer: string;
  tokenUrl: string;
  scopes: string[];
}

export interface ServiceAccountProfileMetadata {
  keyId: string;
  userId: string;
  issuer: string;
  tokenUrl: string;
  scopes: string[];
}

function arrayBufferToPEM(buffer: ArrayBuffer, label: "PUBLIC KEY" | "PRIVATE KEY"): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let offset = 0; offset < bytes.length; offset += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
  }
  const base64 = btoa(binary);
  const lines = base64.match(/.{1,64}/g)?.join("\n") ?? "";
  return `-----BEGIN ${label}-----\n${lines}\n-----END ${label}-----\n`;
}

export async function generateServiceAccountKeyPair(
  webCrypto: Crypto = globalThis.crypto,
): Promise<GeneratedServiceAccountKeyPair> {
  if (!webCrypto?.subtle) {
    throw new Error("Secure browser cryptography is unavailable.");
  }

  const keyPair = await webCrypto.subtle.generateKey(
    {
      name: "RSASSA-PKCS1-v1_5",
      modulusLength: 2048,
      publicExponent: new Uint8Array([1, 0, 1]),
      hash: "SHA-256",
    },
    true,
    ["sign", "verify"],
  );

  const [spki, pkcs8] = await Promise.all([
    webCrypto.subtle.exportKey("spki", keyPair.publicKey),
    webCrypto.subtle.exportKey("pkcs8", keyPair.privateKey),
  ]);

  return {
    publicKeyPem: arrayBufferToPEM(spki, "PUBLIC KEY"),
    privateKeyPem: arrayBufferToPEM(pkcs8, "PRIVATE KEY"),
  };
}

export function createServiceAccountProfile(
  metadata: ServiceAccountProfileMetadata,
  privateKey: string,
): ServiceAccountProfile {
  return {
    type: "serviceaccount",
    keyId: metadata.keyId,
    key: privateKey,
    userId: metadata.userId,
    issuer: metadata.issuer,
    tokenUrl: metadata.tokenUrl,
    scopes: [...metadata.scopes],
  };
}

export function serializeServiceAccountProfile(profile: ServiceAccountProfile): string {
  return `${JSON.stringify(profile, null, 2)}\n`;
}
