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
	equipmentSlots?: SlotInfo[];     // Per-slot resolved loadout (powers IP tooltip + Party panel)
	activeSpellSlots?: SpellSlotInfo[]; // Bound abilities (Q/W/E/Armor/Head/Shoes/Potion/Food)
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
	sessions?: ArchivedSession[]; // Local-disk archived sessions
	zones?: ZoneVisit[];          // Recent zone history
	dungeon?: DungeonRun;         // Active run-scoped scope (when inside a dungeon)
	loot?: LootEntry[];           // Loot ring buffer (most recent first)
	looterTotals?: LooterTotals[];// Per-looter rollup
	visiblePlayers?: VisiblePlayer[]; // Tracked players not in party — "add to party" candidates
}

// VisiblePlayer is an "add to party" candidate: a tracked, named player
// the agent sees but who isn't in the party. Surfaced in the Party panel
// so mixed-guild parties the agent never saw form can be built by hand.
export interface VisiblePlayer {
	userGuid: string;
	name: string;
	itemPower?: number;
	classCode?: string;
	role?: string;
	roleLabel?: string;
}

export interface LootEntry {
	at: string;
	looter: string;
	looterIsLocal?: boolean;
	lootedFrom?: string;
	itemIndex?: number;
	uniqueName?: string;
	displayName?: string;
	quantity: number;
	isSilver?: boolean;
	silverValue?: number;
	zone?: string;
	dungeonId?: string;
}

export interface LooterTotals {
	name: string;
	isLocal?: boolean;
	pickups: number;          // number of OtherGrabbedLoot events
	unitsTotal: number;       // sum of stack quantities for items (not silver piles)
	silverPicked: number;     // direct silver picked off corpses / chests
	silverValueLoot: number;  // AODP-estimated market value of looted items
	topItemName?: string;     // display name of highest-priced single drop
	topItemValue?: number;    // silver value of that top item
	recentItemName?: string;  // display name of most-recent non-silver pickup
	onlySilver?: boolean;     // true when every entry was a silver pile
	lastPickupAt: string;     // ISO timestamp of most-recent pickup
	source?: "local" | "party" | "guild" | "friend";
}

export interface SlotInfo {
	slot: string;     // "MainHand" / "OffHand" / "Head" / "Chest" / "Shoes" / "Bag" / "Cape" / "Mount" / "Potion" / "Food"
	name?: string;
	itemPower?: number;
}

export interface SpellSlotInfo {
	slot: string;     // "Q" / "W" / "E" / "Armor" / "Head" / "Shoes" / "Cape" / "Potion" / "Food"
	name?: string;
}

export interface ArchivedSession {
	id: string;
	startedAt: string;
	endedAt: string;
	durationMs: number;
	zone?: string;
	localName?: string;
	fameTotal: number;
	silverTotal: number;
	respecTotal: number;
	mightTotal: number;
	deathsTotal: number;
	fightCount: number;
	tag?: string;
}

export interface ZoneVisit {
	name: string;
	enteredAt: string;
	leftAt?: string;
	durationMs?: number;
}

export interface DungeonRun {
	id: string;
	zone: string;
	type: string; // "solo" | "group" | "avalonian" | "mists" | "hellgate"
	enteredAt: string;
	endedAt?: string;
	durationMs?: number;
	fameGained: number;
	silverGained: number;
	respecGained: number;
	mightGained: number;
	deathsInRun?: number;
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
// Each pane shows two chips: Current (this-fight bucket) and Total
// (session-wide bucket). DPS / HPS are still rendered in the rate
// column on every row; they're just no longer sortable sub-metrics
// because nobody ranks parties by instantaneous rate.
export type SubMetric =
	| "damageCurrent" | "damageTotal"
	| "healCurrent"   | "healTotal"
	| "takenCurrent"  | "takenTotal";

// SubMetricsByMode tells the UI which sub-metric chips to show per mode,
// and which one is the default. Keep in sync with `primaryForSub` in
// MeterTable.tsx.
export const SUB_METRICS_BY_MODE: Record<Exclude<Mode, "mechanics">, Array<{ id: SubMetric; label: string }>> = {
	damage: [
		{ id: "damageCurrent", label: "Current" },
		{ id: "damageTotal",   label: "Total" },
	],
	heal: [
		{ id: "healCurrent", label: "Current" },
		{ id: "healTotal",   label: "Total" },
	],
	taken: [
		{ id: "takenCurrent", label: "Current" },
		{ id: "takenTotal",   label: "Total" },
	],
};

export const DEFAULT_SUB_METRIC: Record<Exclude<Mode, "mechanics">, SubMetric> = {
	damage: "damageCurrent",
	heal: "healCurrent",
	taken: "takenCurrent",
};
