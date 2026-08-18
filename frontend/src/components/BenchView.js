/**
 * Component: BenchView
 * Astryx Bench & Substitute Priority View
 */

export function renderBenchView(benchPlayers = []) {
  const container = document.createElement('div');
  container.className = 'astryx-bench-container';

  const renderBenchCard = (p, index) => {
    const isGK = p.element_type === 1;
    const priorityLabel = isGK ? 'GK Sub' : `Sub ${index}`;
    const badgeStatus = p.status_badge || 'success';
    const fdrVal = p.fdr || 3;
    const opp = p.opponent_short_name || 'TBD';
    const homeAway = p.is_home ? '(H)' : '(A)';

    return `
      <div class="astryx-bench-slot">
        <span class="astryx-bench-priority">${priorityLabel}</span>
        <div class="astryx-pitch-player" style="background-color: var(--astryx-color-bg-subtle);">
          <span class="astryx-status-dot ${badgeStatus}"></span>
          <div class="astryx-player-shirt">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2L4 5v5c0 5.55 3.84 10.74 8 12 4.16-1.26 8-6.45 8-12V5l-8-3z"/>
            </svg>
          </div>
          <div class="astryx-player-name">${p.web_name || 'Player'}</div>
          <div class="astryx-player-xp" style="color: var(--astryx-color-text-secondary);">${p.projected_xp ? p.projected_xp.toFixed(1) : '0.0'} xP</div>
          <div class="astryx-player-meta">
            <span class="astryx-player-fixture">${opp} ${homeAway}</span>
            <span class="astryx-fdr-badge astryx-fdr-${fdrVal}">${fdrVal}</span>
          </div>
        </div>
      </div>
    `;
  };

  container.innerHTML = `
    <div class="astryx-flex-row" style="justify-content: space-between;">
      <h3 class="astryx-heading-card" style="font-size: var(--astryx-font-size-base);">Dugout & Substitutes</h3>
      <span class="astryx-text-muted">Auto-ordered by replacement value</span>
    </div>
    <div class="astryx-bench-row">
      ${benchPlayers.map(renderBenchCard).join('')}
    </div>
  `;

  return container;
}
