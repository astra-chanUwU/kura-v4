// Minimal WebAuthn browser glue (no dependency): converts the server's
// base64url option values for navigator.credentials and serializes results back.

function b64urlToBuffer(value: string): ArrayBuffer {
  const padded = value
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(Math.ceil(value.length / 4) * 4, "=");
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

function bufferToB64url(buffer: ArrayBuffer | ArrayBufferView): string {
  const bytes =
    buffer instanceof ArrayBuffer
      ? new Uint8Array(buffer)
      : new Uint8Array(buffer.buffer, buffer.byteOffset, buffer.byteLength);
  let binary = "";
  bytes.forEach((b) => (binary += String.fromCharCode(b)));
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

interface CredentialOptions {
  publicKey: Record<string, unknown>;
}

function friendlyPasskeyError(err: unknown): Error {
  const message = err instanceof Error ? err.message : String(err);
  const host = typeof window !== "undefined" ? window.location.hostname : "";
  const isIP = /^\d+\.\d+\.\d+\.\d+$/.test(host) || host.includes(":");
  if (isIP || /invalid domain/i.test(message)) {
    const port =
      typeof window !== "undefined" && window.location.port ? `:${window.location.port}` : "";
    return new Error(
      `Passkeys don't work at an IP address. Open http://localhost${port} instead — it's the same app, and browsers only allow passkeys on a name like localhost.`,
    );
  }
  return err instanceof Error ? err : new Error(message);
}

export function passkeysSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "credentials" in navigator &&
    typeof PublicKeyCredential !== "undefined"
  );
}

export async function createPasskey(options: unknown): Promise<string> {
  const raw = (options as CredentialOptions).publicKey as unknown as {
    challenge: string;
    user: { id: string } & Record<string, unknown>;
    excludeCredentials?: { id: string }[];
  } & Record<string, unknown>;
  const publicKey = {
    ...raw,
    challenge: b64urlToBuffer(raw.challenge),
    user: { ...raw.user, id: b64urlToBuffer(raw.user.id) },
    excludeCredentials: raw.excludeCredentials?.map((c) => ({ ...c, id: b64urlToBuffer(c.id) })),
  } as PublicKeyCredentialCreationOptions;
  let credential: PublicKeyCredential | null;
  try {
    credential = (await navigator.credentials.create({
      publicKey,
    })) as PublicKeyCredential | null;
  } catch (err) {
    throw friendlyPasskeyError(err);
  }
  if (!credential) throw new Error("Passkey setup was cancelled.");
  const response = credential.response as AuthenticatorAttestationResponse;
  return JSON.stringify({
    id: credential.id,
    rawId: bufferToB64url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToB64url(response.attestationObject),
      clientDataJSON: bufferToB64url(response.clientDataJSON),
      transports: typeof response.getTransports === "function" ? response.getTransports() : [],
    },
  });
}

export async function getPasskey(options: unknown): Promise<string> {
  const raw = (options as CredentialOptions).publicKey as unknown as {
    challenge: string;
    allowCredentials?: { id: string }[];
  } & Record<string, unknown>;
  const publicKey = {
    ...raw,
    challenge: b64urlToBuffer(raw.challenge),
    allowCredentials: raw.allowCredentials?.map((c) => ({ ...c, id: b64urlToBuffer(c.id) })),
  } as PublicKeyCredentialRequestOptions;
  let credential: PublicKeyCredential | null;
  try {
    credential = (await navigator.credentials.get({ publicKey })) as PublicKeyCredential | null;
  } catch (err) {
    throw friendlyPasskeyError(err);
  }
  if (!credential) throw new Error("Passkey sign-in was cancelled.");
  const response = credential.response as AuthenticatorAssertionResponse;
  return JSON.stringify({
    id: credential.id,
    rawId: bufferToB64url(credential.rawId),
    type: credential.type,
    response: {
      authenticatorData: bufferToB64url(response.authenticatorData),
      clientDataJSON: bufferToB64url(response.clientDataJSON),
      signature: bufferToB64url(response.signature),
      userHandle: response.userHandle ? bufferToB64url(response.userHandle) : null,
    },
  });
}
