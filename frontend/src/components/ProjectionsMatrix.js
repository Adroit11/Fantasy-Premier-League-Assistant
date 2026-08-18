/**
 * Component: ProjectionsMatrix
 * Multi-Gameweek Projection Table with FDR and expected score badges
 */

export function renderProjectionsMatrix(projectionsData = {}) {
  const container = document.createElement('div');
  container.className = 'astryx-card astryx-projections-table-wrapper';

  const startGW = projectionsData.start_gw || 1;
  const endGW = projectionsData.end_gw || 5;
  const players = projectionsData.players || [];

  const gwHeaders = [];
  for (let gw = startGW; gw <= endGW; gw++) {
    gwHeaders.push(`<th style="text-align: center; padding: var(--astryx-space-2) var(--astryx-space-3);">GW ${gw}</th>`);
  }

  const rows = players.map((p) => {
    const projCells = (p.projections || []).map((pj) => {
      const opp = pj.opponent_short_name || '-';
      const ha = pj.is_home ? '(H)' : '(A)';
      const xp = pj.projected_xp !== undefined ? pj.projected_xp.toFixed(1) : '-';
      const fdr = pj.fdr || 3;

      return `
        <td style="text-align: center; padding: var(--astryx-space-2) var(--astryx-space-3); border-top: 1px solid var(--astryx-color-border-subtle);">
          <div style="font-weight: var(--astryx-font-weight-bold); color: var(--astryx-color-brand-primary); font-size: var(--astryx-font-size-sm);">${xp}</div>
          <div class="astryx-flex-row" style="justify-content: center; gap: 4px; margin-top: 2px;">
            <span style="font-size: 10px; color: var(--astryx-color-text-muted);">${opp} ${ha}</span>
            <span class="astryx-fdr-badge astryx-fdr-${fdr}" style="width: 18px; height: 16px; font-size: 9px;">${fdr}</span>
          </div>
        </td>
      `;
    }).join('');

    return `
      <tr>
        <td style="padding: var(--astryx-space-2) var(--astryx-space-3); border-top: 1px solid var(--astryx-color-border-subtle);">
          <div style="font-weight: var(--astryx-font-weight-semibold); color: var(--astryx-color-text-primary); font-size: var(--astryx-font-size-sm);">${p.web_name}</div>
          <div class="astryx-text-muted" style="font-size: 11px;">${p.team_short_name} • ${p.position_name} • £${p.cost?.toFixed(1)}m</div>
        </td>
        <td style="text-align: center; font-weight: var(--astryx-font-weight-bold); color: var(--astryx-color-brand-secondary); padding: var(--astryx-space-2) var(--astryx-space-3); border-top: 1px solid var(--astryx-color-border-subtle);">
          ${p.total_xp ? p.total_xp.toFixed(1) : '0.0'}
        </td>
        ${projCells}
      </tr>
    `;
  }).join('');

  container.innerHTML = `
    <div class="astryx-card-header">
      <div class="astryx-flex-row">
        <span class="astryx-icon astryx-icon-md astryx-icon-primary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
        </span>
        <h3 class="astryx-heading-card" style="font-size: var(--astryx-font-size-base);">Projected Scores (GW ${startGW} - GW ${endGW})</h3>
      </div>
      <span class="astryx-badge astryx-badge-brand">${players.length} Players</span>
    </div>

    <div style="overflow-x: auto;">
      <table style="width: 100%; border-collapse: collapse; font-family: var(--astryx-font-sans); font-size: var(--astryx-font-size-xs);">
        <thead>
          <tr style="color: var(--astryx-color-text-muted); text-transform: uppercase; font-size: 10px; letter-spacing: var(--astryx-letter-spacing-caps);">
            <th style="text-align: left; padding: var(--astryx-space-2) var(--astryx-space-3);">Player</th>
            <th style="text-align: center; padding: var(--astryx-space-2) var(--astryx-space-3);">Total xP</th>
            ${gwHeaders.join('')}
          </tr>
        </thead>
        <tbody>
          ${rows}
        </tbody>
      </table>
    </div>
  `;

  return container;
}
