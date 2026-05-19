import { classAccent, type RoleKey } from "./format.ts";

interface IPChipProps {
	itemPower?: number;
	classCode?: string;
	roleKey: RoleKey;
	size?: number;
}

// IPChip replaces the old ClassChip in the player row. It surfaces the
// player's averaged Item Power (e.g. 1320) instead of a 3-letter weapon
// abbreviation. The tint still uses the class accent so the row's
// colour identity is preserved.
//
// Falls back to "—" when itemPower is missing or zero (entity registered
// without an equipment event yet).
export function IPChip({ itemPower, classCode, roleKey, size = 28 }: IPChipProps): React.ReactElement {
	const accent = classAccent(classCode, roleKey);
	const display = itemPower && itemPower > 0 ? itemPower.toString() : "—";
	return (
		<div
			title={itemPower && itemPower > 0 ? `Item Power ${itemPower}` : "Item Power unknown"}
			style={{
				minWidth: size,
				height: size,
				padding: "0 6px",
				flexShrink: 0,
				display: "inline-flex",
				alignItems: "center",
				justifyContent: "center",
				border: `1px solid color-mix(in oklab, ${accent} 55%, var(--sk-line-2))`,
				background: `color-mix(in oklab, ${accent} 12%, var(--sk-bg-2))`,
				borderRadius: 3,
				fontFamily: "var(--sk-font-mono)",
				fontSize: 11,
				letterSpacing: "-0.01em",
				color: accent,
				fontWeight: 700,
				fontVariantNumeric: "tabular-nums",
			}}
		>
			{display}
		</div>
	);
}
