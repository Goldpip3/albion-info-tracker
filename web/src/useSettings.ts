import { useCallback, useEffect, useState } from "react";
import type { SubMetric } from "./types.ts";

// Settings is the persisted UI configuration. Stored in localStorage so a
// refresh keeps it. Nothing here goes to the server.
export interface Settings {
	accent: AccentColor;
	density: Density;
	barStyle: BarStyle;
	groupByRole: boolean;
	columns: ColumnVisibility;
	showActivityLog: boolean;
	panes: PaneSet;
	// paneLayout controls how multiple meter panes arrange:
	//   "auto"    — responsive card grid; wraps so panes never crush (default)
	//   "columns" — side-by-side, horizontal-scrolls instead of crushing
	//   "rows"    — full-width stacked cards
	paneLayout: PaneLayout;
	// meterScope sets who shows in the meter + loot + archived fights:
	//   "party"      — confirmed party members + alwaysIncludeNames only
	//   "partyGuild" — + same-guild players (default; dungeons w/ guild)
	//   "everyone"   — every entity with combat activity (ZvZ / open world)
	// Changing it dispatches a `setLootFilter` command so the agent
	// applies the same scope server-side (including when archiving
	// fights, so past-fight playback matches the live view).
	meterScope: MeterScope;
}

export type MeterScope = "party" | "partyGuild" | "everyone";

export type PaneLayout = "auto" | "columns" | "rows";

// PaneSet picks which tables show side-by-side, keyed by sub-metric so
// the SAME metric can appear twice — e.g. "Damage · Current" next to
// "Damage · Session" — letting the user watch the current-fight leader
// and the session leader at once. At least one must stay on; the UI
// blocks turning off the last one.
export type PaneSet = Record<SubMetric, boolean>;

// Accent options match the design canvas's Theme accent row (cyan = local
// default). When changed, the App-level effect overwrites --sk-local on
// :root so every component re-tints without re-rendering.
export type AccentColor = "cyan" | "violet" | "amber" | "lime" | "rose";
export type Density = 28 | 34 | 40;
export type BarStyle = "outline" | "tint" | "solid";

export interface ColumnVisibility {
	dps: boolean;
	hps: boolean;
	damageTaken: boolean;
	healing: boolean;
}

const DEFAULT: Settings = {
	accent: "cyan",
	density: 40,
	barStyle: "outline",
	groupByRole: false,
	meterScope: "partyGuild",
	paneLayout: "auto",
	columns: {
		dps: true,
		hps: false,
		damageTaken: true,
		healing: true,
	},
	showActivityLog: false,
	panes: {
		damageCurrent: true,
		damageTotal:   false,
		healCurrent:   false,
		healTotal:     false,
		takenCurrent:  false,
		takenTotal:    false,
	},
};

// Storage key carries a generation tag. Bumping it (e.g. v1 → v2) forces
// everyone's localStorage back to DEFAULT on the next visit — useful when
// design-token changes shift the visual default (row density, accent set,
// etc.) and we don't want returning users stuck on a stale config.
const KEY = "gda:settings:v2";

export function useSettings(): {
	settings: Settings;
	update: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
	reset: () => void;
} {
	const [settings, setSettings] = useState<Settings>(() => {
		try {
			const raw = localStorage.getItem(KEY);
			if (!raw) return DEFAULT;
			const parsed = JSON.parse(raw) as Partial<Settings>;
			return {
				...DEFAULT,
				...parsed,
				columns: { ...DEFAULT.columns, ...(parsed.columns ?? {}) },
				panes: { ...DEFAULT.panes, ...(parsed.panes ?? {}) },
			};
		} catch {
			return DEFAULT;
		}
	});

	useEffect(() => {
		try {
			localStorage.setItem(KEY, JSON.stringify(settings));
		} catch {
			/* private mode etc. */
		}
	}, [settings]);

	const update = useCallback(
		<K extends keyof Settings>(key: K, value: Settings[K]): void => {
			setSettings((s) => ({ ...s, [key]: value }));
		},
		[],
	);

	const reset = useCallback((): void => setSettings(DEFAULT), []);

	return { settings, update, reset };
}

// accentOklch returns the OKLCH triplet for the local-player accent. The
// "fg" value goes into --sk-local; the "tint" goes into --sk-local-tint
// for outlined bar backgrounds. Hues match the design's swatch row.
export function accentOklch(c: AccentColor): { fg: string; tint: string } {
	switch (c) {
		case "violet": return { fg: "oklch(0.72 0.14 295)", tint: "color-mix(in oklab, oklch(0.72 0.14 295) 14%, transparent)" };
		case "amber":  return { fg: "oklch(0.78 0.13 70)",  tint: "color-mix(in oklab, oklch(0.78 0.13 70) 14%, transparent)" };
		case "lime":   return { fg: "oklch(0.74 0.13 150)", tint: "color-mix(in oklab, oklch(0.74 0.13 150) 14%, transparent)" };
		case "rose":   return { fg: "oklch(0.68 0.18 22)",  tint: "color-mix(in oklab, oklch(0.68 0.18 22) 14%, transparent)" };
		case "cyan":
		default:       return { fg: "oklch(0.78 0.13 215)", tint: "color-mix(in oklab, oklch(0.78 0.13 215) 14%, transparent)" };
	}
}
