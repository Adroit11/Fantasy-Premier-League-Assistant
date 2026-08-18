/**
 * Component: PitchView
 * 2D Interactive Football Pitch with Starting XI & Astryx Tokens
 */

export function renderPitchView(optimalLineup = {}) {
  const container = document.createElement('div');
  container.className = 'astryx-card astryx-pitch-wrapper';
  container.style.padding = 'var(--astryx-space-4)';

  const formation = optimalLineup.formation || '3-4-3';
  const totalXP = optimalLineup.total_projected_xp ? optimalLineup.total_projected_xp.toFixed(1) : '0.0';
  const gks = optimalLineup.goalkeepers || [];
  const defs = optimalLineup.defenders || [];
  const mids = optimalLineup.midfielders || [];
  const fwds = optimalLineup.forwards || [];
  const capId = optimalLineup.captain_id;
  const viceId = optimalLineup.vice_captain_id;

  const renderPlayerCard = (p) => {
    const isCap = p.player_id === capId;
    const isVice = p.player_id === viceId;
    const capClass = isCap ? 'is-captain' : (isVice ? 'is-vice-captain' : '');
    const badgeHTML = isCap 
      ? `<div class="astryx-captain-badge cap">C</div>` 
      : (isVice ? `<div class="astryx-captain-badge vice">V</div>` : '');

    const badgeStatus = p.status_badge || 'success';
    const fdrVal = p.fdr || 3;
    const opp = p.opponent_short_name || 'TBD';
    const homeAway = p.is_home ? '(H)' : '(A)';

    return `
      <div class="astryx-pitch-player ${capClass}" data-player-id="${p.player_id}">
        <span class="astryx-status-dot ${badgeStatus}"></span>
        ${badgeHTML}
        <div class="astryx-player-shirt">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2L4 5v5c0 5.55 3.84 10.74 8 12 4.16-1.26 8-6.45 8-12V5l-8-3z"/>
          </svg>
        </div>
        <div class="astryx-player-name">${p.web_name || 'Player'}</div>
        <div class="astryx-player-xp">${p.projected_xp ? p.projected_xp.toFixed(1) : '0.0'} xP</div>
        <div class="astryx-player-meta">
          <span class="astryx-player-fixture">${opp} ${homeAway}</span>
          <span class="astryx-fdr-badge astryx-fdr-${fdrVal}">${fdrVal}</span>
        </div>
      </div>
    `;
  };

  container.innerHTML = `
    <div class="astryx-card-header" style="margin-bottom: var(--astryx-space-3);">
      <div>
        <h2 class="astryx-heading-card">Optimal Starting XI</h2>
        <p class="astryx-text-secondary">Formation: <strong style="color: var(--astryx-color-brand-primary);">${formation}</strong></p>
      </div>
      <div class="astryx-flex-row">
        <span class="astryx-label-caps">Projected XI Total</span>
        <span class="astryx-stat-value" style="font-size: var(--astryx-font-size-2xl);">${totalXP}</span>
      </div>
    </div>

    <div class="astryx-pitch-container">
      <div class="astryx-pitch-lines"></div>
      <div class="astryx-pitch-center-circle"></div>

      <!-- Forwards Row -->
      <div class="astryx-pitch-row forwards-row">
        ${fwds.map(renderPlayerCard).join('')}
      </div>

      <!-- Midfielders Row -->
      <div class="astryx-pitch-row midfielders-row">
        ${mids.map(renderPlayerCard).join('')}
      </div>

      <!-- Defenders Row -->
      <div class="astryx-pitch-row defenders-row">
        ${defs.map(renderPlayerCard).join('')}
      </div>

      <!-- Goalkeeper Row -->
      <div class="astryx-pitch-row goalkeeper-row">
        ${gks.map(renderPlayerCard).join('')}
      </div>
    </div>
  `;

  return container;
}
