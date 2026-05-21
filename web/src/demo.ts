import { useEffect, useState } from "react";
import type { DailyStat, Snapshot } from "./types.ts";

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

// demoDaily synthesizes a week of per-day economy totals ending today, so
// the Sessions-tab ProgressChart has something to render under ?demo=1.
// Today's bar (last) matches the demo session headline numbers.
function demoDaily(): DailyStat[] {
	const fameK   = [380, 520, 610, 290, 740, 455, 95];   // thousands of fame
	const silverK = [110, 140, 90, 60, 175, 120, 27];     // thousands of silver
	const respec  = [3, 5, 4, 2, 6, 4, 2];
	const might   = [820, 1100, 640, 410, 1320, 760, 148];
	const deaths  = [1, 0, 2, 0, 1, 0, 0];
	const out: DailyStat[] = [];
	for (let i = 6; i >= 0; i--) {
		const d = new Date();
		d.setDate(d.getDate() - i);
		const mm = String(d.getMonth() + 1).padStart(2, "0");
		const dd = String(d.getDate()).padStart(2, "0");
		const idx = 6 - i;
		out.push({
			date: `${d.getFullYear()}-${mm}-${dd}`,
			fameTotal:   fameK[idx]   * 1000 * 10_000,
			silverTotal: silverK[idx] * 1000 * 10_000,
			respecTotal: respec[idx] * 10_000,
			mightTotal:  might[idx]  * 10_000,
			deathsTotal: deaths[idx],
		});
	}
	return out;
}

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
		sessions: [
			{
				id: "demo-1", startedAt: new Date(Date.now() - 86_400_000).toISOString(),
				endedAt: new Date(Date.now() - 86_400_000 + 3_600_000).toISOString(),
				durationMs: 3_600_000, zone: "Mists",
				localName: "Hesper", fightCount: 14,
				fameTotal: 612_000 * 10_000, silverTotal: 84_000 * 10_000,
				respecTotal: 4 * 10_000, mightTotal: 1200 * 10_000, deathsTotal: 1,
			},
			{
				id: "demo-2", startedAt: new Date(Date.now() - 172_800_000).toISOString(),
				endedAt: new Date(Date.now() - 172_800_000 + 5_400_000).toISOString(),
				durationMs: 5_400_000, zone: "Keepers Hide Farm 2",
				localName: "Hesper", fightCount: 22,
				fameTotal: 1_400_000 * 10_000, silverTotal: 320_000 * 10_000,
				respecTotal: 8 * 10_000, mightTotal: 850 * 10_000, deathsTotal: 0,
			},
		],
		daily: demoDaily(),
		zones: [
			{ name: "Caerleon", enteredAt: new Date(Date.now() - 1_800_000).toISOString(), leftAt: new Date(Date.now() - 1_200_000).toISOString(), durationMs: 600_000 },
			{ name: "Mists Hub", enteredAt: new Date(Date.now() - 1_200_000).toISOString(), leftAt: new Date(Date.now() - 600_000).toISOString(), durationMs: 600_000 },
			{ name: "Keepers Hide Farm 2", enteredAt: new Date(Date.now() - 600_000).toISOString(), durationMs: 600_000 },
		],
		dungeon: {
			id: "demo-run", zone: "Keepers Hide Farm 2", type: "group",
			enteredAt: new Date(Date.now() - 540_000).toISOString(),
			durationMs: 540_000,
			fameGained: 78_000 * 10_000,
			silverGained: 12_500 * 10_000,
			respecGained: 0,
			mightGained: 32 * 10_000,
			deathsInRun: 0,
		},
		loot: [
			{ at: new Date(Date.now() - 30_000).toISOString(), looter: "Hesper", looterIsLocal: true, lootedFrom: "Keeper Boss",     uniqueName: "T6_2H_DUALCROSSBOW_CRYSTAL@2", displayName: "Adept's Arclight Blasters", quantity: 1, silverValue: 412000, zone: "Keepers Hide Farm 2" },
			{ at: new Date(Date.now() - 110_000).toISOString(), looter: "Jorah",   lootedFrom: "Keeper Acolyte", uniqueName: "T6_BAG", displayName: "Adept's Bag", quantity: 1, silverValue: 18500, zone: "Keepers Hide Farm 2" },
			{ at: new Date(Date.now() - 170_000).toISOString(), looter: "Orson",   lootedFrom: "Mob",         uniqueName: "T6_ARMOR_PLATE_SET1@1", displayName: "Adept's Plate Armor", quantity: 1, silverValue: 24300, zone: "Keepers Hide Farm 2" },
			{ at: new Date(Date.now() - 230_000).toISOString(), looter: "Hesper", looterIsLocal: true, lootedFrom: "Mob",         isSilver: true, quantity: 3120 },
			{ at: new Date(Date.now() - 290_000).toISOString(), looter: "Jorah",   lootedFrom: "Mob",         uniqueName: "T5_HEAD_LEATHER_SET1", displayName: "Expert's Mercenary Hood", quantity: 1, silverValue: 6200, zone: "Keepers Hide Farm 2" },
		],
		looterTotals: [
			{
				name: "Hesper",
				isLocal: true,
				pickups: 2,
				unitsTotal: 1,
				silverPicked: 3120,
				silverValueLoot: 412000,
				topItemName: "Adept's Arclight Blasters",
				topItemValue: 412000,
				lastPickupAt: new Date(Date.now() - 30_000).toISOString(),
			},
			{
				name: "Jorah",
				pickups: 2,
				unitsTotal: 2,
				silverPicked: 0,
				silverValueLoot: 24700,
				topItemName: "Adept's Bag",
				topItemValue: 18500,
				lastPickupAt: new Date(Date.now() - 110_000).toISOString(),
			},
			{
				name: "Orson",
				pickups: 1,
				unitsTotal: 1,
				silverPicked: 0,
				silverValueLoot: 24300,
				topItemName: "Adept's Plate Armor",
				topItemValue: 24300,
				lastPickupAt: new Date(Date.now() - 170_000).toISOString(),
			},
		],
	};

	return { snapshot: snap, lastMessageAt: Date.now() };
}
