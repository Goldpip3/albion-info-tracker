// Wire types — must stay in sync with agent/internal/domain/snapshot.go
// and agent/internal/push/protocol.go.

export interface PlayerSnapshot {
	userGuid: string;
	objectId?: number;
	name: string;
	guild?: string;
	isLocal?: boolean;
	classCode?: string; // 3-letter chip text ("DGR", "FIR", …) or "—"
	role?: string;      // "T" | "H" | "R" | "M" | "S" | "?"
	roleLabel?: string; // "MELEE DPS · DAGGERS"
	currentDamage: number;
	currentDps: number;
	overallDamage: number;
	overallDps: number;
	currentHeal: number;
	currentHps: number;
	overallHeal: number;
	overallHps: number;
	currentTaken: number;
	overallTaken: number;
}

export interface Composition {
	tank: number;
	healer: number;
	ranged: number;
	melee: number;
	support: number;
	unknown: number;
	total: number;
}

export interface Fight {
	number: number;
	elapsedMs: number;
	inCombat: boolean;
}

export interface Snapshot {
	generatedAt: string;
	players: PlayerSnapshot[];
	composition?: Composition;
	fight?: Fight;
}

export interface SnapshotEnvelope {
	v: number;
	type: "snapshot";
	ts: number;
	snap: Snapshot;
}

export type Envelope = SnapshotEnvelope;

export type ConnectionState = "disconnected" | "connecting" | "connected";

// Display modes correspond to columns in the design.
export type Mode = "damage" | "heal" | "taken" | "mechanics";
