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

// CLASS_ACCENT maps a 3-letter chip code to a specific hex accent. Holy
// Staff (HLY) and Nature Staff (NTR) split here so the two healer weapons
// read distinctly — gold vs green — even though both share RoleHealer.
const CLASS_ACCENT: Record<string, string> = {
	// Healers (split per-weapon)
	HLY: "#ffd770", DVN: "#ffd770", RED: "#ffd770",
	NTR: "#5cf0a4", WLD: "#5cf0a4", FAL: "#5cf0a4",
	// Tanks
	HAM: "#8aa3ff", MAC: "#8aa3ff", QRT: "#8aa3ff", KNK: "#8aa3ff",
	// Ranged DPS
	LBW: "#ffb43a", WBW: "#ffb43a", BOW: "#ffb43a", XBW: "#ffb43a",
	FIR: "#ffb43a", CRS: "#ffb43a",
	// Control
	FRO: "#6fd8ff",
	// Support
	ARC: "#c08cff",
	// Melee DPS
	DGR: "#ff6464", GRT: "#ff6464", SWD: "#ff6464",
	AXE: "#ff6464", SPR: "#ff6464",
};

const ROLE_ACCENT: Record<RoleKey, string> = {
	tank:    "#8aa3ff",
	healer:  "#5cf0a4",
	rdps:    "#ffb43a",
	mdps:    "#ff6464",
	support: "#c08cff",
	control: "#6fd8ff",
	unknown: "#6a6a76",
};

// classAccent picks the per-weapon hex color when known; otherwise falls
// back to the role color. Drives bar fill, DPS column, and class chip.
export function classAccent(classCode: string | undefined, role: RoleKey): string {
	if (classCode && CLASS_ACCENT[classCode]) return CLASS_ACCENT[classCode];
	return ROLE_ACCENT[role];
}

// Pretty-print a spell uniquename. TOKEN_CASE → Title case.
export function prettySpell(name?: string): string {
	if (!name) return "";
	const parts = name.split("_").filter(Boolean);
	if (parts.length === 0) return name;
	const first = parts[0].charAt(0).toUpperCase() + parts[0].slice(1).toLowerCase();
	const rest = parts.slice(1).map((p) => p.toLowerCase()).join(" ");
	return rest ? `${first} ${rest}` : first;
}
