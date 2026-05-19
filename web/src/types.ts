// Wire types — must stay in sync with agent/internal/domain/snapshot.go
// and agent/internal/push/protocol.go.

export interface PlayerSnapshot {
	userGuid: string;
	objectId?: number;
	name: string;
	guild?: string;
	isLocal?: boolean;
	classCode?: string; // 3-letter chip text ("DGR", "FIR", …) or "—" — still
	                    // used internally to tint the bar / DPS column.
	role?: string;      // "T" | "H" | "R" | "M" | "S" | "C" | "?"
	roleLabel?: string; // "DAGGERS" / "ADEPT'S ARCLIGHT BLASTERS"
	itemPower?: number; // Averaged IP across core gear slots — rendered in the
	                    // IPChip in place of the old XBW-style 3-letter chip.
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
	itemPower?: number;
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

// SubMetric narrows what value drives the sort + bar within a Mode.
// WoW Details has the same idea: in the Damage pane, you can flip
// between Damage Done / DPS / Total to re-rank by that field.
export type SubMetric =
	| "damageCurrent" | "damageDps"   | "damageTotal"
	| "healCurrent"   | "healHps"     | "healTotal" | "healOverheal"
	| "takenCurrent"  | "takenTotal";

// SubMetricsByMode tells the UI which sub-metric chips to show per mode,
// and which one is the default. Keep in sync with `primaryForSub` in
// MeterTable.tsx.
export const SUB_METRICS_BY_MODE: Record<Exclude<Mode, "mechanics">, Array<{ id: SubMetric; label: string }>> = {
	damage: [
		{ id: "damageCurrent", label: "Damage" },
		{ id: "damageDps",     label: "DPS" },
		{ id: "damageTotal",   label: "Total" },
	],
	heal: [
		{ id: "healCurrent",  label: "Healing" },
		{ id: "healHps",      label: "HPS" },
		{ id: "healTotal",    label: "Total" },
		{ id: "healOverheal", label: "Overheal" },
	],
	taken: [
		{ id: "takenCurrent", label: "Taken" },
		{ id: "takenTotal",   label: "Total" },
	],
};

export const DEFAULT_SUB_METRIC: Record<Exclude<Mode, "mechanics">, SubMetric> = {
	damage: "damageCurrent",
	heal: "healCurrent",
	taken: "takenCurrent",
};
