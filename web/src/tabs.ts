import type { Snapshot } from "./types.ts";
import type { Tab } from "./TabBar.tsx";

export type TabId = "meter" | "loot" | "party" | "sessions";

// tabsFor builds the four primary navigation entries with live counts
// drawn from the current snapshot. Pages that don't have a snapshot
// (e.g. before pairing) pass null and the counts collapse to zero,
// which the TabBar hides.
//
// The token is forwarded as ?token=<…> so a click navigates with the
// pairing intact even when localStorage isn't shared (other browser,
// other profile). Adding the param costs nothing when localStorage is
// already populated; LootPage / PartyPage / SessionsPage strip the
// param on mount.
export function tabsFor(snapshot: Snapshot | null, token: string): Tab[] {
	const t = token.trim();
	const q = t ? `?token=${encodeURIComponent(t)}` : "";
	return [
		{ id: "meter",    label: "Meter",    href: `/${q}` },
		{ id: "loot",     label: "Loot",     href: `/loot${q}`,     count: snapshot?.loot?.length ?? 0 },
		{ id: "party",    label: "Party",    href: `/party${q}`,    count: snapshot?.players?.length ?? 0 },
		{ id: "sessions", label: "Sessions", href: `/sessions${q}`, count: snapshot?.sessions?.length ?? 0 },
	];
}
