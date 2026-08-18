/**
 * Component: AvailabilityBanner
 * Displays squad injury news, doubtful players, and press conference quotes
 */

export function renderAvailabilityBanner(data = {}) {
  const container = document.createElement('div');
  container.className = 'astryx-card astryx-availability-wrapper';

  const alerts = data.alerts || [];
  const injuredCount = data.injured_count || 0;
  const doubtCount = data.doubt_count || 0;

  if (alerts.length === 0) {
    container.innerHTML = `
      <div class="astryx-flex-row" style="gap: var(--astryx-space-3);">
        <span class="astryx-icon astryx-icon-md astryx-icon-success">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </span>
        <div>
          <h3 class="astryx-heading-card" style="font-size: var(--astryx-font-size-base); color: var(--astryx-color-status-success);">Squad Fully Fit</h3>
          <p class="astryx-text-secondary">No active injuries, suspensions, or rotation doubts flagged in your squad.</p>
        </div>
      </div>
    `;
    return container;
  }

  const alertItems = alerts.map((a) => {
    const badgeClass = `astryx-badge-${a.severity || 'warning'}`;
    const chanceLabel = a.chance_of_playing !== null ? `${a.chance_of_playing}% chance` : 'Uncertain';

    return `
      <div style="background: var(--astryx-color-bg-surface-raised); border: 1px solid var(--astryx-color-border-subtle); border-radius: var(--astryx-radius-sm); padding: var(--astryx-space-3); margin-top: var(--astryx-space-2);">
        <div class="astryx-flex-row" style="justify-content: space-between; margin-bottom: var(--astryx-space-1);">
          <div class="astryx-flex-row">
            <strong style="color: var(--astryx-color-text-primary); font-size: var(--astryx-font-size-sm);">${a.web_name}</strong>
            <span class="astryx-text-muted">${a.team_short_name} • ${a.position_name}</span>
          </div>
          <span class="astryx-badge ${badgeClass}">${chanceLabel}</span>
        </div>
        <p style="font-size: var(--astryx-font-size-xs); color: var(--astryx-color-text-secondary); line-height: 1.4;">
          ${a.news || 'Status flag updated by Premier League'}
        </p>
      </div>
    `;
  }).join('');

  container.innerHTML = `
    <div class="astryx-card-header" style="margin-bottom: var(--astryx-space-2);">
      <div class="astryx-flex-row">
        <span class="astryx-icon astryx-icon-md astryx-icon-warning">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </span>
        <h3 class="astryx-heading-card" style="font-size: var(--astryx-font-size-base);">Availability Alerts (${alerts.length})</h3>
      </div>
      <div class="astryx-flex-row">
        ${injuredCount > 0 ? `<span class="astryx-badge astryx-badge-danger">${injuredCount} Out</span>` : ''}
        ${doubtCount > 0 ? `<span class="astryx-badge astryx-badge-warning">${doubtCount} Doubt</span>` : ''}
      </div>
    </div>
    <div class="astryx-alert-list">
      ${alertItems}
    </div>
  `;

  return container;
}
