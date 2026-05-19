import { useCallback, useEffect, useState } from "react";

// Settings is the persisted UI configuration. Stored in localStorage so a
// refresh keeps it. Nothing here goes to the server.
export interface Settings {
	accent: AccentColor;
	density: Density;
	barStyle: BarStyle;
	pinLocal: boolean;
	groupByRole: boolean; // hooked up in T3
	columns: ColumnVisibility;
}

export type AccentColor = "amber" | "blue" | "purple" | "green" | "red" | "yellow" | "pink";
export type Density = 24 | 28 | 34;
export type BarStyle = "outline" | "tint" | "solid";

export interface ColumnVisibility {
	itemPower: boolean;
	dps: boolean;
	hps: boolean;
	damageTaken: boolean;
	healing: boolean;
}

const DEFAULT: Settings = {
	accent: "amber",
	density: 28,
	barStyle: "outline",
	pinLocal: true,
	groupByRole: false,
	columns: { itemPower: false, dps: true, hps: false, damageTaken: true, healing: true },
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

// accentOklch returns the OKLCH triplet for an accent so we can drive the
// CSS variable in App.tsx. Matching tones to the design's color row.
export function accentOklch(c: AccentColor): { fg: string; dim: string } {
	switch (c) {
		case "blue":   return { fg: "oklch(0.75 0.15 240)", dim: "oklch(0.6 0.13 240)" };
		case "purple": return { fg: "oklch(0.7 0.18 295)",  dim: "oklch(0.55 0.16 295)" };
		case "green":  return { fg: "oklch(0.75 0.18 145)", dim: "oklch(0.6 0.16 145)" };
		case "red":    return { fg: "oklch(0.7 0.2 25)",    dim: "oklch(0.55 0.18 25)" };
		case "yellow": return { fg: "oklch(0.85 0.17 95)",  dim: "oklch(0.7 0.15 95)" };
		case "pink":   return { fg: "oklch(0.75 0.2 350)",  dim: "oklch(0.6 0.18 350)" };
		case "amber":
		default:       return { fg: "oklch(0.79 0.16 78)",  dim: "oklch(0.65 0.13 78)" };
	}
}
