/**
 * Component: TransferAdvisor
 * Displays AI prioritized transfer recommendations
 */

export function renderTransferAdvisor(transfersData = {}) {
  const container = document.createElement('div');
  container.className = 'astryx-card astryx-transfers-wrapper';

  const recs = transfersData.recommendations || [];
  const bank = transfersData.bank_available !== undefined ? `£${transfersData.bank_available.toFixed(1)}m` : '£0.0m';

  if (recs.length === 0) {
    container.innerHTML = `
      <div class="astryx-card-header">
        <h3 class="astryx-heading-card" style="font-size: var(--astryx-font-size-base);">Transfer Recommendations</h3>
        <span class="astryx-text-muted">Bank: ${bank}</span>
      </div>
      <p class="astryx-text-secondary">Your squad is in great shape! No immediate high-value transfer targets outperform your current starters within budget.</p>
    `;
    return container;
  }

  const recCards = recs.map((r) => {
    const costSign = r.cost_difference > 0 ? `+£${r.cost_difference.toFixed(1)}m` : `£${r.cost_difference.toFixed(1)}m`;

    return `
      <div style="background: var(--astryx-color-bg-surface-raised); border: 1px solid var(--astryx-color-border-subtle); border-radius: var(--astryx-radius-sm); padding: var(--astryx-space-3); margin-bottom: var(--astryx-space-2);">
        <div class="astryx-flex-row" style="justify-content: space-between; margin-bottom: var(--astryx-space-2);">
          <span class="astryx-badge astryx-badge-cyan">${r.position_name}</span>
          <span class="astryx-badge astryx-badge-brand">+${r.net_xp_gain.toFixed(1)} Net xP</span>
        </div>

        <div style="display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: var(--astryx-space-2); margin-bottom: var(--astryx-space-2);">
          <!-- Out -->
          <div style="background: rgba(233, 0, 82, 0.08); border: 1px solid rgba(233, 0, 82, 0.2); border-radius: var(--astryx-radius-xs); padding: var(--astryx-space-2);">
            <span class="astryx-label-caps" style="color: var(--astryx-color-status-danger);">SELL OUT</span>
            <div style="font-weight: var(--astryx-font-weight-semibold); color: var(--astryx-color-text-primary); font-size: var(--astryx-font-size-sm);">${r.player_out_name}</div>
            <div class="astryx-text-muted" style="font-size: 11px;">${r.player_out_team} • £${r.player_out_cost.toFixed(1)}m • ${r.player_out_xp.toFixed(1)} xP</div>
          </div>

          <!-- Arrow -->
          <div style="color: var(--astryx-color-brand-primary); font-weight: bold; text-align: center;">➜</div>

          <!-- In -->
          <div style="background: rgba(0, 255, 135, 0.08); border: 1px solid rgba(0, 255, 135, 0.2); border-radius: var(--astryx-radius-xs); padding: var(--astryx-space-2);">
            <span class="astryx-label-caps" style="color: var(--astryx-color-brand-primary);">BUY IN</span>
            <div style="font-weight: var(--astryx-font-weight-semibold); color: var(--astryx-color-text-primary); font-size: var(--astryx-font-size-sm);">${r.player_in_name}</div>
            <div class="astryx-text-muted" style="font-size: 11px;">${r.player_in_team} • £${r.player_in_cost.toFixed(1)}m • ${r.player_in_xp.toFixed(1)} xP</div>
          </div>
        </div>

        <div class="astryx-flex-row" style="justify-content: space-between; font-size: 11px;">
          <span class="astryx-text-secondary">${r.reason}</span>
          <span class="astryx-text-muted">Cost Delta: ${costSign}</span>
        </div>
      </div>
    `;
  }).join('');

  container.innerHTML = `
    <div class="astryx-card-header" style="margin-bottom: var(--astryx-space-2);">
      <div class="astryx-flex-row">
        <span class="astryx-icon astryx-icon-md astryx-icon-primary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <polyline points="17 1 21 5 17 9"/>
            <path d="M3 11V9a4 4 0 0 1 4-4h14"/>
            <polyline points="7 23 3 19 7 15"/>
            <path d="M21 13v2a4 4 0 0 1-4 4H3"/>
          </svg>
        </span>
        <h3 class="astryx-heading-card" style="font-size: var(--astryx-font-size-base);">Transfer Suggestions</h3>
      </div>
      <span class="astryx-text-muted">Bank: <strong style="color: var(--astryx-color-brand-secondary);">${bank}</strong></span>
    </div>
    <div class="astryx-transfer-list">
      ${recCards}
    </div>
  `;

  return container;
}
