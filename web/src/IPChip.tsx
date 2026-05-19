import { useRef, useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { classAccent, type RoleKey } from "./format.ts";
import type { SlotInfo } from "./types.ts";

interface IPChipProps {
	itemPower?: number;
	classCode?: string;
	roleKey: RoleKey;
	size?: number;
	slots?: SlotInfo[];
}

// IPChip replaces the old 3-letter weapon chip with the player's
// averaged Item Power. Tinted by the class accent so the row's colour
// identity is preserved.
//
// Hovering reveals a per-slot breakdown ("MainHand 1340 · Cape 1280 …")
// portaled to document.body so it doesn't clip inside narrow panes.
// Slots with no item are omitted so the tooltip stays compact.
export function IPChip({ itemPower, classCode, roleKey, size = 28, slots }: IPChipProps): React.ReactElement {
	const accent = classAccent(classCode, roleKey);
	const display = itemPower && itemPower > 0 ? itemPower.toString() : "—";
	const chipRef = useRef<HTMLDivElement>(null);
	const [anchor, setAnchor] = useState<{ left: number; top: number } | null>(null);

	const showTooltip = slots && slots.length > 0;

	useEffect(() => {
		if (!anchor || !chipRef.current) return;
		// Reposition on scroll/resize so the floating tooltip tracks
		// the chip even when the meter is scrolled inside a pane.
		const reposition = (): void => {
			const r = chipRef.current?.getBoundingClientRect();
			if (r) setAnchor({ left: r.right + 8, top: r.top });
		};
		window.addEventListener("scroll", reposition, true);
		window.addEventListener("resize", reposition);
		return () => {
			window.removeEventListener("scroll", reposition, true);
			window.removeEventListener("resize", reposition);
		};
	}, [anchor]);

	return (
		<>
			<div
				ref={chipRef}
				onMouseEnter={() => {
					if (!showTooltip || !chipRef.current) return;
					const r = chipRef.current.getBoundingClientRect();
					setAnchor({ left: r.right + 8, top: r.top });
				}}
				onMouseLeave={() => setAnchor(null)}
				title={!showTooltip && itemPower ? `Item Power ${itemPower}` : undefined}
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
					cursor: showTooltip ? "help" : "default",
				}}
			>
				{display}
			</div>
			{anchor && showTooltip && createPortal(
				<div
					style={{
						position: "fixed",
						left: anchor.left,
						top: anchor.top,
						minWidth: 200,
						background: "var(--sk-bg-2)",
						border: "1px solid var(--sk-line-2)",
						borderRadius: 6,
						padding: "8px 10px",
						boxShadow: "0 12px 32px -8px rgba(0,0,0,0.6)",
						zIndex: 1000,
						pointerEvents: "none",
					}}
				>
					<div className="flex items-baseline" style={{ gap: 8, marginBottom: 6, paddingBottom: 6, borderBottom: "1px solid var(--sk-line)" }}>
						<span className="sk-upper" style={{ fontSize: 9.5, color: "var(--sk-fg-3)" }}>Item Power · avg</span>
						<span className="sk-mono" style={{ fontSize: 13, fontWeight: 700, color: accent, marginLeft: "auto" }}>
							{itemPower ?? 0}
						</span>
					</div>
					<div className="flex flex-col" style={{ gap: 3 }}>
						{(slots ?? []).filter((s) => s.slot !== "Bag" && s.slot !== "Mount" && s.slot !== "Potion" && s.slot !== "Food").map((s) => (
							<div key={s.slot} className="flex items-baseline justify-between" style={{ gap: 8 }}>
								<span className="sk-upper" style={{ fontSize: 9, color: "var(--sk-fg-3)" }}>{s.slot}</span>
								<span style={{ fontSize: 11, color: "var(--sk-fg-1)", marginLeft: 8 }}>{s.name || "—"}</span>
								<span className="sk-mono" style={{ fontSize: 11, color: accent, fontWeight: 600, marginLeft: "auto" }}>
									{s.itemPower ?? 0}
								</span>
							</div>
						))}
					</div>
				</div>,
				document.body,
			)}
		</>
	);
}
