// Shared number / time / role formatters. Mirrors sim.jsx's helpers from
// the design handoff so the visual output matches what was prototyped.

export function fmt(n: number | null | undefined): string {
	if (n == null || isNaN(n as number)) return "—";
	const v = n as number;
	if (v >= 1_000_000) return (v / 1_000_000).toFixed(2) + "M";
	if (v >= 10_000)    return (v / 1000).toFixed(0)      + "K";
	if (v >= 1000)      return (v / 1000).toFixed(1)      + "K";
	return String(Math.round(v));
}

export function fmtRate(n: number | null | undefined): string {
	if (n == null || isNaN(n as number)) return "—";
	const v = n as number;
	if (v >= 10_000) return (v / 1000).toFixed(0) + "k";
	if (v >= 1000)   return (v / 1000).toFixed(1) + "k";
	return String(Math.round(v));
}

export function fmtDuration(sec: number): string {
	if (sec <= 0) return "0:00";
	const m = Math.floor(sec / 60);
	const s = Math.floor(sec % 60);
	return `${m}:${s.toString().padStart(2, "0")}`;
}

// rankBy returns a comparator that ranks players by a metric descending
// with userGuid as a stable tiebreak. The Go agent rebuilds the player
// list from a randomized map each 250ms tick, so without the tiebreak
// that random order leaks through Array.sort whenever the leaders tie
// (everyone at 0 out of combat, healers/tanks tied at 0 damage, equal
// totals) and the displayed "top"/"carried by" name flickers. Route
// every player ranking through this so no sort site can forget it.
export function rankBy<T extends { userGuid: string }>(
	selector: (p: T) => number,
): (a: T, b: T) => number {
	return (a, b) => {
		const d = (selector(b) || 0) - (selector(a) || 0);
		if (d !== 0) return d;
		return a.userGuid.localeCompare(b.userGuid);
	};
}

// roleKey maps the agent's T/H/R/M/S/C/? letter to the design's lowercase
// role token used to look up CSS role colors (var(--sk-role-tank), etc.).
export type RoleKey = "tank" | "healer" | "rdps" | "mdps" | "support" | "control" | "unknown";

export function roleKeyOf(role: string | undefined): RoleKey {
	switch (role) {
		case "T": return "tank";
		case "H": return "healer";
		case "R": return "rdps";
		case "M": return "mdps";
		case "S": return "support";
		case "C": return "control";
		default:  return "unknown";
	}
}

// CLASS_ACCENT maps a 3-letter chip code to a specific hex accent. The
// palette is tuned for maximum separation on the near-black background —
// each role sits on a distinct hue (azure / gold / emerald / orange /
// crimson / teal / violet) so a glance tells roles apart. Holy Staff (HLY)
// and Nature Staff (NTR) still split — gold vs emerald — even though both
// share RoleHealer.
const CLASS_ACCENT: Record<string, string> = {
	// Healers (split per-weapon: Holy = gold, Nature = emerald)
	HLY: "#ffd24a", DVN: "#ffd24a", RED: "#ffd24a",
	NTR: "#36d989", WLD: "#36d989", FAL: "#36d989",
	// Tanks — azure blue
	HAM: "#4f8cff", MAC: "#4f8cff", QRT: "#4f8cff", KNK: "#4f8cff",
	// Ranged DPS — orange
	LBW: "#ff7a2e", WBW: "#ff7a2e", BOW: "#ff7a2e", XBW: "#ff7a2e",
	FIR: "#ff7a2e", CRS: "#ff7a2e",
	// Control (Frost) — teal/cyan
	FRO: "#3ad1e0",
	// Support (Arcane) — violet
	ARC: "#a877ff",
	// Melee DPS — crimson/rose
	DGR: "#ff4d6d", GRT: "#ff4d6d", SWD: "#ff4d6d",
	AXE: "#ff4d6d", SPR: "#ff4d6d",
};

const ROLE_ACCENT: Record<RoleKey, string> = {
	tank:    "#4f8cff",
	healer:  "#36d989",
	rdps:    "#ff7a2e",
	mdps:    "#ff4d6d",
	support: "#a877ff",
	control: "#3ad1e0",
	unknown: "#7a7a88",
};

// classAccent picks the per-weapon hex color when known; otherwise falls
// back to the role color. Drives bar fill, DPS column, and class chip.
export function classAccent(classCode: string | undefined, role: RoleKey): string {
	if (classCode && CLASS_ACCENT[classCode]) return CLASS_ACCENT[classCode];
	return ROLE_ACCENT[role];
}

// Pretty-print a spell uniquename. The agent does the heavy lifting now —
// localization + override + Title-Case prettifier — so by the time names
// reach the web they're already in human form ("Caltrops", "Auto Attack",
// "Flickershot"). This function exists for one case: an old agent build
// that still ships raw "CROSSBOW_FLICKERSHOT_E" uniquenames over the
// wire. Names without underscores pass through unchanged so we don't
// mangle "Auto Attack" into "Auto attack".
export function prettySpell(name?: string): string {
	if (!name) return "";
	if (!name.includes("_")) return name; // already clean, leave it alone
	const parts = name.split("_").filter(Boolean);
	if (parts.length === 0) return name;
	const first = parts[0].charAt(0).toUpperCase() + parts[0].slice(1).toLowerCase();
	const rest = parts.slice(1).map((p) => p.toLowerCase()).join(" ");
	return rest ? `${first} ${rest}` : first;
}
