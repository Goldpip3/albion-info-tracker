import type { RoleKey } from "./format.ts";

interface ClassChipProps {
	code: string;
	roleKey: RoleKey;
	size?: number;
}

// 3-letter monospace class glyph in a bordered square, tinted by role.
// Mirrors meter.jsx ClassChip from the design.
export function ClassChip({ code, roleKey, size = 22 }: ClassChipProps): React.ReactElement {
	const roleColor = `var(--sk-role-${roleKey})`;
	return (
		<div
			style={{
				width: size,
				height: size,
				flexShrink: 0,
				display: "inline-flex",
				alignItems: "center",
				justifyContent: "center",
				border: `1px solid color-mix(in oklab, ${roleColor} 55%, var(--sk-line-2))`,
				background: `color-mix(in oklab, ${roleColor} 10%, var(--sk-bg-2))`,
				borderRadius: 3,
				fontFamily: "var(--sk-font-mono)",
				fontSize: 9,
				letterSpacing: "0.05em",
				color: roleColor,
				fontWeight: 600,
			}}
		>
			{code || "—"}
		</div>
	);
}
