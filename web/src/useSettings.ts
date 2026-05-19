import { useCallback, useEffect, useState } from "react";

// Settings is the persisted UI configuration. Stored in localStorage so a
// refresh keeps it. Nothing here goes to the server.
export interface Settings {
	accent: AccentColor;
	density: Density;
	barStyle: BarStyle;
	pinLocal: boolean;
	groupByRole: boolean;
	columns: ColumnVisibility;
}

// Accent options match the design canvas's Theme accent row (cyan = local
// default). When changed, the App-level effect overwrites --sk-local on
// :root so every component re-tints without re-rendering.
export type AccentColor = "cyan" | "violet" | "amber" | "lime" | "rose";
export type Density = 24 | 28 | 34;
export type BarStyle = "outline" | "tint" | "solid";

export interface ColumnVisibility {
	itemPower: boolean;
	dps: boolean;
	hps: boolean;
	damageTaken: boolean;
	healing: boolean;
	critPct: boolean;
}

const DEFAULT: Settings = {
	accent: "cyan",
	density: 28,
	barStyle: "outline",
	pinLocal: true,
	groupByRole: false,
	columns: {
		itemPower: false,
		dps: true,
		hps: false,
		damageTaken: true,
		healing: true,
		critPct: false,
	},
};

const KEY = "skirmish:settings";

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
