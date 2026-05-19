import { classAccent, type RoleKey } from "./format.ts";

interface ClassChipProps {
	code: string;
	roleKey: RoleKey;
	size?: number;
}

// ClassChip is a 22–28px bordered square showing a 3-letter weapon
// abbreviation tinted by the per-weapon accent. Mirrors meter.jsx's
// ClassChip from the design — 12%/55% mixes on bg-2 / line-2 so the
// chip's tint reads as "of that family" without overpowering the row.
export function ClassChip({ code, roleKey, size = 22 }: ClassChipProps): React.ReactElement {
	const accent = classAccent(code, roleKey);
	return (
		<div
			title={code || "—"}
			style={{
				width: size,
				height: size,
				flexShrink: 0,
				display: "inline-flex",
				alignItems: "center",
				justifyContent: "center",
				border: `1px solid color-mix(in oklab, ${accent} 55%, var(--sk-line-2))`,
				background: `color-mix(in oklab, ${accent} 12%, var(--sk-bg-2))`,
				borderRadius: 3,
				fontFamily: "var(--sk-font-mono)",
				fontSize: 9,
				letterSpacing: "0.05em",
				color: accent,
				fontWeight: 700,
			}}
		>
			{code || "—"}
		</div>
	);
}
