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

	deaths?: number;
	overheal?: number;

	spells?: SpellBreakdown[];
	sessionSpells?: SpellBreakdown[];
	targets?: TargetBreakdown[];
	activeEffects?: number[];
	assists?: AssistBreakdown[];
}

export interface AssistBreakdown {
	index: number;
	name?: string;
	uptimeMs: number;
	damageDuring: number;
}

export interface SpellBreakdown {
	index: number;
	name?: string;
	totalDamage: number;
	maxHit: number;
	hits: number;
	casts?: number;
}

export interface TargetBreakdown {
	objectId: number;
	name?: string;
	damage: number;
}

export interface Composition {
	tank: number;
	healer: number;
	ranged: number;
	melee: number;
	support: number;
	control: number;
	unknown: number;
	total: number;
}

export interface Fight {
	number: number;
	elapsedMs: number;
	inCombat: boolean;
	zone?: string;
}

export interface Session {
	startedAt: string;
	elapsedMs: number;
	fameTotal: number;
	silverTotal: number; // FixPoint internal, divide by 10_000 in UI
	respecTotal: number; // FixPoint internal, divide by 10_000 in UI
	mightTotal: number;  // FixPoint internal, divide by 10_000 in UI
	deathsTotal: number;
}

export interface ActivityEvent {
	ts: number;
	kind: "hit" | "heal" | "death";
	actor?: string;
	actorIsLocal?: boolean;
	target?: string;
	amount?: number;
	spellName?: string;
	bigHit?: boolean;
}

export interface Snapshot {
	generatedAt: string;
	players: PlayerSnapshot[];
	composition?: Composition;
	fight?: Fight;
	session?: Session;
	events?: ActivityEvent[];
	recent?: FightArchive[];
}

export interface FightArchive {
	number: number;
	startedAt: string;
	endedAt: string;
	durationMs: number;
	players: FightPlayerArchive[];
}

export interface FightPlayerArchive {
	userGuid: string;
	name: string;
	classCode?: string;
	role?: string;
	roleLabel?: string;
	isLocal?: boolean;
	damage: number;
	dps: number;
	heal: number;
	hps: number;
	overheal?: number;
	taken: number;
	deaths?: number;
	spells?: SpellBreakdown[];
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
