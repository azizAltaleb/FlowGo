import { webcrypto } from "node:crypto";
import { describe, expect, it } from "vitest";
import {
  createServiceAccountProfile,
  generateServiceAccountKeyPair,
  serializeServiceAccountProfile,
} from "./serviceAccount";

describe("service-account credentials", () => {
  it("generates a 2048-bit RSA SPKI public key and PKCS8 private key", async () => {
    const crypto = webcrypto as unknown as Crypto;
    const pair = await generateServiceAccountKeyPair(crypto);

    expect(pair.publicKeyPem).toMatch(
      /^-----BEGIN PUBLIC KEY-----\n(?:[A-Za-z0-9+/=]{1,64}\n)+-----END PUBLIC KEY-----\n$/,
    );
    expect(pair.privateKeyPem).toMatch(
      /^-----BEGIN PRIVATE KEY-----\n(?:[A-Za-z0-9+/=]{1,64}\n)+-----END PRIVATE KEY-----\n$/,
    );

    const publicKey = await crypto.subtle.importKey(
      "spki",
      pemBytes(pair.publicKeyPem),
      { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" },
      true,
      ["verify"],
    );
    expect((publicKey.algorithm as RsaKeyAlgorithm).modulusLength).toBe(2048);

    await expect(
      crypto.subtle.importKey(
        "pkcs8",
        pemBytes(pair.privateKeyPem),
        { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" },
        false,
        ["sign"],
      ),
    ).resolves.toBeDefined();
  });

  it("assembles the downloadable profile without changing private material", () => {
    const profile = createServiceAccountProfile(
      {
        keyId: "key-1",
        userId: "user-1",
        issuer: "https://iam.example.com",
        tokenUrl: "https://iam.example.com/oauth/v2/token",
        scopes: ["openid", "urn:zitadel:iam:org:project:id:project-1:aud"],
      },
      "PRIVATE PEM",
    );

    expect(JSON.parse(serializeServiceAccountProfile(profile))).toEqual({
      type: "serviceaccount",
      keyId: "key-1",
      key: "PRIVATE PEM",
      userId: "user-1",
      issuer: "https://iam.example.com",
      tokenUrl: "https://iam.example.com/oauth/v2/token",
      scopes: ["openid", "urn:zitadel:iam:org:project:id:project-1:aud"],
    });
  });
});

function pemBytes(pem: string): Uint8Array<ArrayBuffer> {
  const base64 = pem.replace(/-----[^-]+-----|\s/g, "");
  return Uint8Array.from(atob(base64), (character) => character.charCodeAt(0));
}
