interface EmptyStateProps {
	title: string;
	body: string;
	icon?: boolean;
}

// EmptyState is the standard "nothing to show yet" panel used by the
// Party / Loot / Sessions bodies when the snapshot has no data for the
// view. A faint GDA glyph + title + sub-line reads as "real product"
// rather than a bare one-liner.
export function EmptyState({ title, body, icon = true }: EmptyStateProps): React.ReactElement {
	return (
		<div
			className="flex flex-col items-center justify-center"
			style={{
				padding: 48,
				textAlign: "center",
				flex: 1,
				gap: 18,
				color: "var(--sk-fg-2)",
			}}
		>
			{icon && (
				<img
					src="/assets/icon-GDA-128.png"
					srcSet="/assets/icon-GDA-64.png 1x, /assets/icon-GDA-128.png 2x, /assets/icon-GDA-256.png 4x"
					alt=""
					width={64}
					height={64}
					style={{
						display: "block",
						borderRadius: 12,
						opacity: 0.12,
						filter: "grayscale(1)",
					}}
				/>
			)}
			<div className="flex flex-col items-center" style={{ gap: 6, maxWidth: 380 }}>
				<div className="sk-upper" style={{ color: "var(--sk-fg-1)", fontSize: 12, fontWeight: 700, letterSpacing: "0.08em" }}>
					{title}
				</div>
				<div style={{ fontSize: 12.5, color: "var(--sk-fg-3)", lineHeight: 1.55 }}>
					{body}
				</div>
			</div>
		</div>
	);
}
