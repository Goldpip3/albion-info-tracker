import type { AccentColor, BarStyle, Density, PaneMode, Settings } from "./useSettings.ts";
import { accentOklch } from "./useSettings.ts";

interface SettingsPanelProps {
	settings: Settings;
	update: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
	reset: () => void;
	onClose: () => void;
}

const ACCENT_OPTIONS: AccentColor[] = ["cyan", "violet", "amber", "lime", "rose"];
const DENSITY_OPTIONS: Array<[Density, string]> = [[24, "Compact"], [28, "Regular"], [34, "Comfy"]];
const BAR_STYLE_OPTIONS: Array<[BarStyle, string]> = [["outline", "Outline"], ["tint", "Tint"], ["solid", "Solid"]];

export function SettingsPanel({ settings, update, reset, onClose }: SettingsPanelProps): React.ReactElement {
	return (
		<div
			className="fixed inset-0 z-50 flex items-center justify-center p-4"
			style={{ background: "rgba(0,0,0,0.6)", backdropFilter: "blur(4px)" }}
			onClick={onClose}
		>
			<div
				className="w-full max-w-lg flex flex-col"
				style={{
					background: "var(--sk-bg-0)",
					border: "1px solid var(--sk-line)",
					borderRadius: 8,
					maxHeight: "85vh",
					overflow: "hidden",
				}}
				onClick={(e) => e.stopPropagation()}
			>
				<header
					className="flex items-center justify-between px-4 py-3"
					style={{ borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
				>
					<div className="flex items-center" style={{ gap: 10 }}>
						<span style={{ fontSize: 14, fontWeight: 600 }}>Settings</span>
						<span
							className="sk-upper pl-2.5"
							style={{ borderLeft: "1px solid var(--sk-line)", color: "var(--sk-fg-3)" }}
						>
							meter.skirmish.gg
						</span>
					</div>
					<button
						onClick={onClose}
						aria-label="Close"
						style={{
							appearance: "none",
							border: "1px solid var(--sk-line)",
							background: "var(--sk-bg-2)",
							color: "var(--sk-fg-1)",
							width: 24,
							height: 24,
							borderRadius: 4,
							cursor: "pointer",
							fontSize: 12,
						}}
					>
						✕
					</button>
				</header>

				<div className="overflow-auto" style={{ padding: "16px 18px", flex: 1 }}>
					<Group title="Display">
						<Setting
							label="Columns"
							sub="Choose which secondary metrics show on each row"
						>
							<div className="flex flex-wrap" style={{ gap: 6 }}>
								<ColChip on={settings.columns.itemPower} onClick={() => update("columns", { ...settings.columns, itemPower: !settings.columns.itemPower })}>Item Power</ColChip>
								<ColChip on={settings.columns.dps}       onClick={() => update("columns", { ...settings.columns, dps: !settings.columns.dps })}>DPS</ColChip>
								<ColChip on={settings.columns.hps}       onClick={() => update("columns", { ...settings.columns, hps: !settings.columns.hps })}>HPS</ColChip>
								<ColChip on={settings.columns.damageTaken} onClick={() => update("columns", { ...settings.columns, damageTaken: !settings.columns.damageTaken })}>Damage Taken</ColChip>
								<ColChip on={settings.columns.healing}   onClick={() => update("columns", { ...settings.columns, healing: !settings.columns.healing })}>Healing</ColChip>
								<ColChip on={settings.columns.critPct}   onClick={() => update("columns", { ...settings.columns, critPct: !settings.columns.critPct })}>Crit %</ColChip>
							</div>
						</Setting>
						<Setting label="Theme accent" sub="Color used for your local-player row">
							<div className="inline-flex" style={{ gap: 5 }}>
								{ACCENT_OPTIONS.map((c) => (
									<button
										key={c}
										onClick={() => update("accent", c)}
										aria-label={c}
										style={{
											width: 22,
											height: 22,
											borderRadius: 4,
											background: accentOklch(c).fg,
											border: settings.accent === c ? "2px solid var(--sk-fg-0)" : "1px solid var(--sk-line-2)",
											boxShadow: settings.accent === c ? `0 0 0 2px color-mix(in oklab, ${accentOklch(c).fg} 35%, transparent)` : "none",
											cursor: "pointer",
											padding: 0,
										}}
									/>
								))}
							</div>
						</Setting>
						<Setting label="Density" sub="Row height">
							<Segment
								options={DENSITY_OPTIONS}
								current={settings.density}
								onChange={(v) => update("density", v)}
							/>
						</Setting>
						<Setting label="Bar style">
							<Segment
								options={BAR_STYLE_OPTIONS}
								current={settings.barStyle}
								onChange={(v) => update("barStyle", v)}
							/>
						</Setting>
					</Group>

					<Group title="Layout">
						<Setting label="Pane mode" sub="Single tab-switched view, or three columns side by side">
							<Segment<PaneMode>
								options={[["single", "Single"], ["triple", "Triple"]]}
								current={settings.paneMode}
								onChange={(v) => update("paneMode", v)}
							/>
						</Setting>
						<Setting label="Activity log" sub="Show recent hits / heals / deaths under the meter">
							<Toggle
								on={settings.showActivityLog}
								onClick={() => update("showActivityLog", !settings.showActivityLog)}
							/>
						</Setting>
					</Group>

					<Group title="Sort & grouping">
						<Setting label="Group by role" sub="Cluster Tanks, Healers, DPS visually">
							<Toggle on={settings.groupByRole} onClick={() => update("groupByRole", !settings.groupByRole)} />
						</Setting>
						<Setting label="Pin local user" sub="Always show your row, even if outside top N">
							<Toggle on={settings.pinLocal} onClick={() => update("pinLocal", !settings.pinLocal)} />
						</Setting>
					</Group>
				</div>

				<footer
					className="flex items-center justify-between px-4 py-2.5"
					style={{ borderTop: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
				>
					<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>Changes auto-save</span>
					<div className="flex" style={{ gap: 8 }}>
						<button onClick={reset} style={ghostBtn}>Reset defaults</button>
						<button onClick={onClose} style={primaryBtn}>Done</button>
					</div>
				</footer>
			</div>
		</div>
	);
}

function Group({ title, children }: { title: string; children: React.ReactNode }): React.ReactElement {
	return (
		<div style={{ marginBottom: 20 }}>
			<div
				className="sk-upper"
				style={{ marginBottom: 8, fontWeight: 600, color: "var(--sk-fg-2)" }}
			>
				{title}
			</div>
			<div
				style={{
					background: "var(--sk-bg-1)",
					border: "1px solid var(--sk-line)",
					borderRadius: 6,
					overflow: "hidden",
				}}
			>
				{children}
			</div>
		</div>
	);
}

function Setting({
	label,
	sub,
	children,
}: {
	label: string;
	sub?: string;
	children: React.ReactNode;
}): React.ReactElement {
	return (
		<div
			className="grid items-center"
			style={{
				gridTemplateColumns: "1fr auto",
				gap: 14,
				padding: "10px 14px",
				borderBottom: "1px solid var(--sk-line)",
			}}
		>
			<div>
				<div style={{ fontSize: 12.5, color: "var(--sk-fg-0)", fontWeight: 500 }}>{label}</div>
				{sub && <div style={{ fontSize: 11, color: "var(--sk-fg-2)", marginTop: 2 }}>{sub}</div>}
			</div>
			<div>{children}</div>
		</div>
	);
}

function ColChip({
	on,
	onClick,
	children,
}: {
	on: boolean;
	onClick: () => void;
	children: React.ReactNode;
}): React.ReactElement {
	return (
		<button
			onClick={onClick}
			className="inline-flex items-center"
			style={{
				gap: 5,
				padding: "3px 8px",
				borderRadius: 99,
				fontSize: 11,
				color: on ? "var(--sk-fg-0)" : "var(--sk-fg-3)",
				background: on ? "color-mix(in oklab, var(--sk-local) 12%, var(--sk-bg-2))" : "var(--sk-bg-2)",
				border: on
					? "1px solid color-mix(in oklab, var(--sk-local) 45%, var(--sk-line))"
					: "1px solid var(--sk-line)",
				cursor: "pointer",
				appearance: "none",
			}}
		>
			<span
				style={{
					width: 6,
					height: 6,
					borderRadius: 99,
					background: on ? "var(--sk-local)" : "var(--sk-fg-3)",
				}}
			/>
			{children}
		</button>
	);
}

interface SegmentProps<T extends string | number> {
	options: Array<[T, string]>;
	current: T;
	onChange: (v: T) => void;
}

function Segment<T extends string | number>({ options, current, onChange }: SegmentProps<T>): React.ReactElement {
	return (
		<div
			className="inline-flex"
			style={{
				padding: 2,
				gap: 1,
				background: "var(--sk-bg-2)",
				border: "1px solid var(--sk-line)",
				borderRadius: 5,
			}}
		>
			{options.map(([k, label]) => {
				const active = k === current;
				return (
					<button
						key={String(k)}
						onClick={() => onChange(k)}
						style={{
							appearance: "none",
							border: 0,
							padding: "4px 9px",
							borderRadius: 3,
							fontSize: 11,
							color: active ? "var(--sk-fg-0)" : "var(--sk-fg-2)",
							background: active ? "var(--sk-bg-3)" : "transparent",
							fontWeight: active ? 500 : 400,
							cursor: "pointer",
						}}
					>
						{label}
					</button>
				);
			})}
		</div>
	);
}

function Toggle({ on, onClick }: { on: boolean; onClick: () => void }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			aria-pressed={on}
			style={{
				appearance: "none",
				display: "inline-flex",
				alignItems: "center",
				width: 28,
				height: 16,
				padding: 2,
				background: on ? "var(--sk-local)" : "var(--sk-line-2)",
				borderRadius: 99,
				border: 0,
				cursor: "pointer",
				transition: "background 200ms var(--sk-ease)",
			}}
		>
			<span
				style={{
					width: 12,
					height: 12,
					borderRadius: 99,
					background: "var(--sk-fg-0)",
					transform: on ? "translateX(12px)" : "translateX(0)",
					transition: "transform 180ms var(--sk-ease)",
				}}
			/>
		</button>
	);
}

const primaryBtn: React.CSSProperties = {
	appearance: "none",
	border: "1px solid var(--sk-line-2)",
	background: "var(--sk-bg-3)",
	color: "var(--sk-fg-0)",
	padding: "6px 12px",
	borderRadius: 4,
	cursor: "pointer",
	fontSize: 11.5,
	fontWeight: 500,
};

const ghostBtn: React.CSSProperties = {
	appearance: "none",
	border: "1px solid var(--sk-line)",
	background: "transparent",
	color: "var(--sk-fg-1)",
	padding: "6px 11px",
	borderRadius: 4,
	cursor: "pointer",
	fontSize: 11.5,
};
