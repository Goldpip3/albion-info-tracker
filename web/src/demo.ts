import { useEffect, useState } from "react";
import type { Snapshot } from "./types.ts";

// useDemoSnapshot returns a hand-crafted Snapshot that exercises every
// per-class accent (daggers, fire staff, frost staff, hammer, holy,
// nature, arcane) and steadily advances combat / session counters so
// the UI animates realistically. Activates when ?demo=1 is in the URL.
// Used to verify the meter visually without needing a live Albion
// session or even the agent running.

export function isDemoMode(): boolean {
	try {
		return new URLSearchParams(window.location.search).get("demo") === "1";
	} catch {
		return false;
	}
}

interface DemoPlayer {
	guid: string;
	name: string;
	classCode: string;
	role: string;
	roleLabel: string;
	itemPower: number;
	dps: number;     // current DPS to tween toward
	taken: number;   // accumulated session taken
	heal: number;    // accumulated session heal
	isLocal?: boolean;
}

const ROSTER: DemoPlayer[] = [
	{ guid: "01", name: "Nyla",        classCode: "DGR", role: "M", roleLabel: "DAGGERS",     itemPower: 1340, dps: 6600, taken: 115000, heal: 0 },
	{ guid: "02", name: "Lirien",      classCode: "DGR", role: "M", roleLabel: "DAGGERS",     itemPower: 1310, dps: 6900, taken: 115000, heal: 0 },
	{ guid: "03", name: "Hesper",      classCode: "DGR", role: "M", roleLabel: "DAGGERS",     itemPower: 1320, dps: 6400, taken: 113000, heal: 0, isLocal: true },
	{ guid: "04", name: "Aldric",      classCode: "FIR", role: "R", roleLabel: "FIRE STAFF",  itemPower: 1290, dps: 4800, taken:  51000, heal: 0 },
	{ guid: "05", name: "Kestrel",     classCode: "FRO", role: "C", roleLabel: "FROST STAFF", itemPower: 1280, dps: 3200, taken:  45000, heal: 0 },
	{ guid: "06", name: "Mervyn",      classCode: "FRO", role: "C", roleLabel: "FROST STAFF", itemPower: 1310, dps: 4000, taken:  45000, heal: 0 },
	{ guid: "07", name: "Fenwick",     classCode: "HAM", role: "T", roleLabel: "HAMMER",      itemPower: 1360, dps: 2100, taken: 270000, heal: 0 },
	{ guid: "08", name: "Edda",        classCode: "ARC", role: "S", roleLabel: "ARCANE STAFF",itemPower: 1300, dps: 2100, taken:  65000, heal: 1130000 },
	{ guid: "09", name: "Jorah",       classCode: "NTR", role: "H", roleLabel: "NATURE STAFF",itemPower: 1330, dps:  789, taken:  90000, heal: 4570000 },
	{ guid: "10", name: "Orson",       classCode: "HLY", role: "H", roleLabel: "HOLY STAFF",  itemPower: 1350, dps:  900, taken:  88000, heal: 3920000 },
];

export function useDemoSnapshot(): { snapshot: Snapshot; lastMessageAt: number } {
	const [t, setT] = useState(15); // fight elapsed seconds
	const startedAt = useState(() => new Date(Date.now() - 18 * 60 * 1000).toISOString())[0];

	useEffect(() => {
		const id = setInterval(() => setT((s) => s + 1), 1000);
		return () => clearInterval(id);
	}, []);

	const players = ROSTER.map((p) => {
		// Add ±5% jitter on dps so sparkline trends and rank ordering feel alive.
		const jitter = (Math.sin((Date.now() / 800) + p.guid.charCodeAt(0)) + 1) / 2;
		const curDps = p.dps * (0.85 + jitter * 0.3);
		const curDamage = Math.round(curDps * t);
		const sessionDamage = Math.round(curDamage * 1.55);
		const curHps = p.heal > 0 ? p.heal / (18 * 60) : 0;
		const curHeal = Math.round(curHps * t);
		return {
			userGuid: p.guid,
			name: p.name,
			classCode: p.classCode,
			role: p.role,
			roleLabel: p.roleLabel,
			itemPower: p.itemPower,
			isLocal: p.isLocal,
			currentDamage: curDamage,
			currentDps: curDps,
			overallDamage: sessionDamage,
			overallDps: curDps * 0.85,
			currentHeal: curHeal,
			currentHps: curHps,
			overallHeal: p.heal,
			overallHps: p.heal / (18 * 60),
			currentTaken: Math.round(p.taken / 8),
			overallTaken: p.taken,
			deaths: 0,
		};
	});

	const composition = ROSTER.reduce(
		(acc, p) => {
			switch (p.role) {
				case "T": acc.tank++; break;
				case "H": acc.healer++; break;
				case "R": acc.ranged++; break;
				case "M": acc.melee++; break;
				case "S": acc.support++; break;
				case "C": acc.control++; break;
			}
			acc.total++;
			return acc;
		},
		{ tank: 0, healer: 0, ranged: 0, melee: 0, support: 0, control: 0, unknown: 0, total: 0 },
	);

	const sessionElapsedMs = 18 * 60 * 1000; // 18-min farm session
	const snap: Snapshot = {
		generatedAt: new Date().toISOString(),
		players,
		composition,
		fight: { number: 4, elapsedMs: t * 1000, inCombat: true, zone: "Keepers Hide Farm 2" },
		session: {
			startedAt,
			elapsedMs: sessionElapsedMs,
			// All values are FixPoint internal units (10_000 = 1 real)
			fameTotal:   95_000  * 10_000, // 95K fame
			silverTotal: 27_000  * 10_000, // 27K silver
			respecTotal: 2       * 10_000, // 2 respec credits
			mightTotal:  148     * 10_000, // 148 might
			deathsTotal: 0,
		},
		events: [],
		recent: [],
	};

	return { snapshot: snap, lastMessageAt: Date.now() };
}
