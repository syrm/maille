export function Header() {
	return (
		<div class="header anim d1">
			<div>
				<h1>Tableau de bord</h1>
				<p>Dernière mise à jour : 6 février 2026, 14:32</p>
			</div>
			<div class="header-actions">
				<div class="period-sel">
					<button class="period-btn">7J</button>
					<button class="period-btn active">1M</button>
					<button class="period-btn">3M</button>
					<button class="period-btn">1A</button>
					<button class="period-btn">Max</button>
				</div>
				<button class="btn-icon" title="Exporter">
					<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
				</button>
			</div>
		</div>
	);
}
