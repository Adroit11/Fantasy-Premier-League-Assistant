/**
 * FPL Assistant 2026/2027 Main Application Orchestrator
 * Fully adhering to Single Action File Pattern & Astryx Design System
 */

import { connectTeamAction } from './actions/connectTeam.action.js';
import { suggestLineupAction } from './actions/suggestLineup.action.js';
import { getAvailabilityNewsAction } from './actions/getAvailabilityNews.action.js';
import { getProjectionsAction } from './actions/getProjections.action.js';
import { suggestTransfersAction } from './actions/suggestTransfers.action.js';

import { renderHeaderBar } from './components/HeaderBar.js';
import { renderPitchView } from './components/PitchView.js';
import { renderBenchView } from './components/BenchView.js';
import { renderAvailabilityBanner } from './components/AvailabilityBanner.js';
import { renderProjectionsMatrix } from './components/ProjectionsMatrix.js';
import { renderTransferAdvisor } from './components/TransferAdvisor.js';

// Application State
const state = {
  teamId: null,
  gameweek: 1,
  teamOverview: null,
  optimalLineup: null,
  availability: null,
  projections: null,
  transfers: null,
  activeTab: 'pitch',
  loading: false,
};

// DOM Elements
const connectForm = document.getElementById('connect-form');
const teamIdInput = document.getElementById('team-id-input');
const connectBtn = document.getElementById('connect-btn');
const demoBtn = document.getElementById('demo-btn');
const headerContainer = document.getElementById('header-container');
const tabContent = document.getElementById('tab-content');
const navTabs = document.querySelectorAll('.astryx-nav-tab');

// Mock fallback for standalone browser preview / testing
function getDemoState() {
  return {
    manager_name: "Alex Ferguson",
    team_name: "Red Devils 26/27",
    overall_rank: 14205,
    total_points: 1420,
    current_gameweek: 1,
    bank: 1.5,
    team_value: 103.5,
    optimal_lineup: {
      formation: "3-4-3",
      total_projected_xp: 68.4,
      captain_id: 13,
      captain_name: "Haaland",
      vice_captain_id: 8,
      vice_captain_name: "Salah",
      goalkeepers: [
        { player_id: 1, web_name: "Raya", element_type: 1, projected_xp: 5.5, fdr: 2, opponent_short_name: "WOL", is_home: true, status_badge: "success" }
      ],
      defenders: [
        { player_id: 3, web_name: "Gabriel", element_type: 2, projected_xp: 6.2, fdr: 2, opponent_short_name: "WOL", is_home: true, status_badge: "success" },
        { player_id: 4, web_name: "Saliba", element_type: 2, projected_xp: 5.8, fdr: 2, opponent_short_name: "WOL", is_home: true, status_badge: "success" },
        { player_id: 5, web_name: "Gvardiol", element_type: 2, projected_xp: 5.4, fdr: 3, opponent_short_name: "CHE", is_home: false, status_badge: "success" }
      ],
      midfielders: [
        { player_id: 8, web_name: "Salah", element_type: 3, projected_xp: 8.8, fdr: 2, opponent_short_name: "IPS", is_home: false, status_badge: "success" },
        { player_id: 9, web_name: "Saka", element_type: 3, projected_xp: 7.5, fdr: 2, opponent_short_name: "WOL", is_home: true, status_badge: "success" },
        { player_id: 10, web_name: "Palmer", element_type: 3, projected_xp: 8.1, fdr: 3, opponent_short_name: "MCI", is_home: true, status_badge: "success" },
        { player_id: 11, web_name: "Rogers", element_type: 3, projected_xp: 5.2, fdr: 2, opponent_short_name: "WHU", is_home: false, status_badge: "success" }
      ],
      forwards: [
        { player_id: 13, web_name: "Haaland", element_type: 4, projected_xp: 9.4, fdr: 3, opponent_short_name: "CHE", is_home: false, status_badge: "success" },
        { player_id: 14, web_name: "Watkins", element_type: 4, projected_xp: 6.8, fdr: 2, opponent_short_name: "WHU", is_home: false, status_badge: "success" },
        { player_id: 15, web_name: "Isak", element_type: 4, projected_xp: 7.0, fdr: 2, opponent_short_name: "SOU", is_home: true, status_badge: "success" }
      ],
      bench: [
        { player_id: 2, web_name: "Valdimarsson", element_type: 1, projected_xp: 1.5, fdr: 3, opponent_short_name: "CRY", is_home: true, status_badge: "success" },
        { player_id: 6, web_name: "Konsa", element_type: 2, projected_xp: 3.8, fdr: 2, opponent_short_name: "WHU", is_home: false, status_badge: "success" },
        { player_id: 7, web_name: "Barco", element_type: 2, projected_xp: 2.1, fdr: 4, opponent_short_name: "ARS", is_home: false, status_badge: "warning" },
        { player_id: 12, web_name: "Winks", element_type: 3, projected_xp: 2.5, fdr: 4, opponent_short_name: "TOT", is_home: true, status_badge: "success" }
      ]
    },
    availability: {
      total_players_checked: 15,
      doubt_count: 1,
      injured_count: 1,
      alerts: [
        { web_name: "Barco", team_short_name: "BHA", position_name: "DEF", status: "d", chance_of_playing: 75, severity: "warning", news: "Knock in training - 75% chance" },
        { web_name: "Shaw", team_short_name: "MUN", position_name: "DEF", status: "i", chance_of_playing: 0, severity: "danger", news: "Calf strain - Expected back Sep 2026" }
      ]
    },
    projections: {
      start_gw: 1,
      end_gw: 5,
      players: [
        {
          web_name: "Haaland", team_short_name: "MCI", position_name: "FWD", cost: 15.0, total_xp: 44.5,
          projections: [
            { opponent_short_name: "CHE", is_home: false, fdr: 3, projected_xp: 9.4 },
            { opponent_short_name: "IPS", is_home: true, fdr: 1, projected_xp: 11.2 },
            { opponent_short_name: "WHU", is_home: false, fdr: 2, projected_xp: 8.5 },
            { opponent_short_name: "BRE", is_home: true, fdr: 2, projected_xp: 8.9 },
            { opponent_short_name: "ARS", is_home: true, fdr: 4, projected_xp: 6.5 }
          ]
        },
        {
          web_name: "Salah", team_short_name: "LIV", position_name: "MID", cost: 12.5, total_xp: 41.0,
          projections: [
            { opponent_short_name: "IPS", is_home: false, fdr: 2, projected_xp: 8.8 },
            { opponent_short_name: "BRE", is_home: true, fdr: 2, projected_xp: 9.0 },
            { opponent_short_name: "MUN", is_home: false, fdr: 3, projected_xp: 7.2 },
            { opponent_short_name: "NFO", is_home: true, fdr: 2, projected_xp: 8.5 },
            { opponent_short_name: "BOU", is_home: true, fdr: 2, projected_xp: 7.5 }
          ]
        },
        {
          web_name: "Saka", team_short_name: "ARS", position_name: "MID", cost: 10.0, total_xp: 36.8,
          projections: [
            { opponent_short_name: "WOL", is_home: true, fdr: 2, projected_xp: 7.5 },
            { opponent_short_name: "AVL", is_home: false, fdr: 3, projected_xp: 6.8 },
            { opponent_short_name: "BHA", is_home: true, fdr: 2, projected_xp: 7.6 },
            { opponent_short_name: "TOT", is_home: false, fdr: 4, projected_xp: 6.9 },
            { opponent_short_name: "MCI", is_home: false, fdr: 5, projected_xp: 5.0 }
          ]
        }
      ]
    },
    transfers: {
      bank_available: 1.5,
      recommendations: [
        {
          position_name: "DEF",
          player_out_name: "Barco",
          player_out_team: "BHA",
          player_out_cost: 4.0,
          player_out_xp: 2.1,
          player_in_name: "Robinson",
          player_in_team: "FUL",
          player_in_cost: 4.5,
          player_in_xp: 4.8,
          net_xp_gain: 2.7,
          cost_difference: 0.5,
          reason: "Favorable fixture run (LEI, IPS) and guaranteed starting minutes"
        },
        {
          position_name: "MID",
          player_out_name: "Winks",
          player_out_team: "LEI",
          player_out_cost: 4.5,
          player_out_xp: 2.5,
          player_in_name: "Minteh",
          player_in_team: "BHA",
          player_in_cost: 5.5,
          player_in_xp: 5.4,
          net_xp_gain: 2.9,
          cost_difference: 1.0,
          reason: "Explosive attacking xG/xA stats in pre-season"
        }
      ]
    }
  };
}

async function loadSquadData(teamId) {
  state.loading = true;
  state.teamId = teamId;
  connectBtn.innerHTML = '<span class="astryx-loading-spinner"></span> Connecting...';
  connectBtn.disabled = true;

  try {
    if (window.fplAPI && window.fplAPI.invokeAction) {
      // Live Go Backend Execution via Electron IPC
      const [overview, lineup, avail, projs, trans] = await Promise.all([
        connectTeamAction(teamId),
        suggestLineupAction(teamId),
        getAvailabilityNewsAction(teamId),
        getProjectionsAction(teamId),
        suggestTransfersAction(teamId),
      ]);

      state.teamOverview = overview;
      state.optimalLineup = lineup.optimal_lineup;
      state.availability = avail;
      state.projections = projs;
      state.transfers = trans;
      state.gameweek = overview.current_gameweek || 1;
    } else {
      // Fallback demo data
      const demo = getDemoState();
      state.teamOverview = demo;
      state.optimalLineup = demo.optimal_lineup;
      state.availability = demo.availability;
      state.projections = demo.projections;
      state.transfers = demo.transfers;
      state.gameweek = demo.current_gameweek;
    }

    renderApp();
  } catch (error) {
    console.error('Failed to load squad:', error);
    alert(`Could not connect team ID ${teamId}: ${error.message}\nLoading Demo Squad for preview.`);
    const demo = getDemoState();
    state.teamOverview = demo;
    state.optimalLineup = demo.optimal_lineup;
    state.availability = demo.availability;
    state.projections = demo.projections;
    state.transfers = demo.transfers;
    renderApp();
  } finally {
    state.loading = false;
    connectBtn.innerHTML = 'Connect Squad';
    connectBtn.disabled = false;
  }
}

function renderApp() {
  if (!state.teamOverview) return;

  // 1. Render Header
  headerContainer.innerHTML = '';
  headerContainer.appendChild(renderHeaderBar(state.teamOverview));

  // 2. Render Active Tab View
  tabContent.innerHTML = '';

  switch (state.activeTab) {
    case 'pitch':
      if (state.optimalLineup) {
        tabContent.appendChild(renderPitchView(state.optimalLineup));
        tabContent.appendChild(renderBenchView(state.optimalLineup.bench));
      }
      break;

    case 'availability':
      if (state.availability) {
        tabContent.appendChild(renderAvailabilityBanner(state.availability));
      }
      break;

    case 'projections':
      if (state.projections) {
        tabContent.appendChild(renderProjectionsMatrix(state.projections));
      }
      break;

    case 'transfers':
      if (state.transfers) {
        tabContent.appendChild(renderTransferAdvisor(state.transfers));
      }
      break;
  }
}

// Event Listeners
connectForm.addEventListener('submit', (e) => {
  e.preventDefault();
  const id = teamIdInput.value.trim();
  if (id) {
    loadSquadData(Number(id));
  }
});

demoBtn.addEventListener('click', () => {
  teamIdInput.value = '12345';
  loadSquadData(12345);
});

navTabs.forEach((tab) => {
  tab.addEventListener('click', () => {
    navTabs.forEach((t) => t.classList.remove('active'));
    tab.classList.add('active');
    state.activeTab = tab.dataset.tab;
    renderApp();
  });
});

// Auto-load team on initial launch (from env or fallback demo ID)
const defaultTeamId = import.meta.env?.VITE_DEFAULT_TEAM_ID ? Number(import.meta.env.VITE_DEFAULT_TEAM_ID) : 12345;
if (teamIdInput) {
  teamIdInput.value = defaultTeamId;
}
loadSquadData(defaultTeamId);
