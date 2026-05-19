import { useEffect } from "react";

// Tab is one entry in the top-level navigation. Routes are simple
// pathnames; clicks intercept the navigation and use history.pushState
// so the page never fully reloads. App listens for the popstate event
// and re-renders the right body in place.
export interface Tab {
	id: string;
	label: string;
	href: string;
	count?: number;
}

interface TabBarProps {
	tabs: Tab[];
	active: string;
}

// softNav swaps the URL and notifies App without triggering a real
// navigation. The synthetic popstate event matches the one the browser
// would fire on a back/forward, so a single listener in App handles
// both directions consistently.
function softNav(href: string): void {
	if (typeof window === "undefined") return;
	window.history.pushState({}, "", href);
	window.dispatchEvent(new PopStateEvent("popstate"));
}

// TabBar renders the row of tabs sitting under the main header. Tabs
// share the row evenly via flex:1 so they aren't bunched left. Active
// tab gets a 2-px bottom border in the damage accent. Counts appear
// after the label when non-zero. document.title syncs to the active
// tab.
export function TabBar({ tabs, active }: TabBarProps): React.ReactElement {
	const activeTab = tabs.find((t) => t.id === active) ?? tabs[0];

	useEffect(() => {
		if (typeof document === "undefined" || !activeTab) return;
		document.title = `GDA · ${activeTab.label}`;
	}, [activeTab]);

	return (
		<nav
			className="flex items-stretch"
			style={{
				height: 38,
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
				flexShrink: 0,
			}}
			aria-label="Primary navigation"
		>
			{tabs.map((t) => {
				const isActive = t.id === active;
				return (
					<a
						key={t.id}
						href={t.href}
						onClick={(e) => {
							// Let Ctrl/Cmd/middle-click open in a new tab the
							// regular way. Otherwise intercept and soft-nav.
							if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return;
							e.preventDefault();
							if (!isActive) softNav(t.href);
						}}
						className="sk-upper inline-flex items-center justify-center"
						style={{
							flex: 1,
							textAlign: "center",
							textDecoration: "none",
							padding: "0 14px",
							gap: 6,
							color: isActive ? "var(--sk-fg-0)" : "var(--sk-fg-2)",
							fontSize: 12.5,
							fontWeight: 600,
							letterSpacing: "0.08em",
							borderBottom: isActive ? "2px solid var(--sk-damage)" : "2px solid transparent",
							transition: "color 160ms var(--sk-ease), border-color 160ms var(--sk-ease)",
							whiteSpace: "nowrap",
						}}
						onMouseEnter={(e) => {
							if (!isActive) (e.currentTarget as HTMLAnchorElement).style.color = "var(--sk-fg-0)";
						}}
						onMouseLeave={(e) => {
							if (!isActive) (e.currentTarget as HTMLAnchorElement).style.color = "var(--sk-fg-2)";
						}}
					>
						{t.label}
						{t.count != null && t.count > 0 && (
							<span
								className="sk-mono"
								style={{
									fontSize: 10,
									color: isActive ? "var(--sk-damage)" : "var(--sk-fg-3)",
									fontWeight: 600,
									letterSpacing: 0,
								}}
							>
								· {t.count}
							</span>
						)}
					</a>
				);
			})}
		</nav>
	);
}
