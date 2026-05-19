// Derive a stable room identifier from a bearer token.
//
// We don't ship any user database — whoever holds a token controls the
// room. Two clients presenting the same token share a room (your agent on
// PC + your browser on phone). Tokens are opaque blobs the user generates
// themselves and configures on both the agent and the website.
//
// The hash is SHA-256 of the token, truncated to 16 hex chars (64 bits).
// That's enough collision resistance for personal use; if multiple users
// somehow collide on the same prefix, change the truncation length.

export async function roomIdFromToken(token: string): Promise<string> {
	const enc = new TextEncoder().encode(token);
	const digest = await crypto.subtle.digest("SHA-256", enc);
	const bytes = new Uint8Array(digest);
	let hex = "";
	for (let i = 0; i < 8; i++) {
		hex += bytes[i].toString(16).padStart(2, "0");
	}
	return hex;
}

// extractToken returns the bearer token from either the Authorization
// header or a ?token= query param. Returns null if absent.
export function extractToken(req: Request): string | null {
	const auth = req.headers.get("Authorization");
	if (auth) {
		const m = /^Bearer\s+(.+)$/i.exec(auth);
		if (m) return m[1];
	}
	const url = new URL(req.url);
	const q = url.searchParams.get("token");
	return q ?? null;
}
