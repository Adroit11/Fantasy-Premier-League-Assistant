/**
 * Component: HeaderBar
 * Astryx Token Compliant Manager & Status Header
 */

export function renderHeaderBar(data = {}) {
  const container = document.createElement('header');
  container.className = 'astryx-card astryx-header-bar';

  const manager = data.manager_name || 'FPL Manager';
  const team = data.team_name || 'My Fantasy XI';
  const rank = data.overall_rank ? data.overall_rank.toLocaleString() : '---';
  const points = data.total_points ?? '---';
  const bank = data.bank !== undefined ? `£${data.bank.toFixed(1)}m` : '£0.0m';
  const gw = data.current_gameweek || 1;

  container.innerHTML = `
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: var(--astryx-space-4);">
      <div class="astryx-flex-row" style="gap: var(--astryx-space-3);">
        <div class="astryx-player-shirt" style="background: var(--astryx-color-brand-surface); border: 1px solid var(--astryx-color-brand-primary);">
          <span class="astryx-icon astryx-icon-md astryx-icon-primary">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
            </svg>
          </span>
        </div>
        <div>
          <div class="astryx-flex-row">
            <h1 class="astryx-heading-card" style="font-size: var(--astryx-font-size-xl);">${team}</h1>
            <span class="astryx-badge astryx-badge-brand">GW ${gw}</span>
          </div>
          <p class="astryx-text-secondary">${manager} • Season 2026/2027</p>
        </div>
      </div>

      <div style="display: flex; gap: var(--astryx-space-6); align-items: center;">
        <div style="text-align: right;">
          <span class="astryx-label-caps">Overall Points</span>
          <div class="astryx-heading-card" style="color: var(--astryx-color-brand-primary); font-size: var(--astryx-font-size-xl);">${points}</div>
        </div>
        <div style="text-align: right;">
          <span class="astryx-label-caps">Global Rank</span>
          <div class="astryx-heading-card" style="font-size: var(--astryx-font-size-xl);">${rank}</div>
        </div>
        <div style="text-align: right;">
          <span class="astryx-label-caps">In The Bank</span>
          <div class="astryx-heading-card" style="color: var(--astryx-color-brand-secondary); font-size: var(--astryx-font-size-xl);">${bank}</div>
        </div>
      </div>
    </div>
  `;

  return container;
}
