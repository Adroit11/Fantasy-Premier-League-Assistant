/**
 * Action: getProjectionsAction
 * Calculates multi-gameweek projected scores (xP)
 */

export async function getProjectionsAction(teamId = null, playerIds = null, startGW = null, endGW = null) {
  const payload = {};
  if (teamId) {
    payload.team_id = Number(teamId);
  }
  if (playerIds && Array.isArray(playerIds)) {
    payload.player_ids = playerIds.map(Number);
  }
  if (startGW) {
    payload.start_gw = Number(startGW);
  }
  if (endGW) {
    payload.end_gw = Number(endGW);
  }

  try {
    const response = await window.fplAPI.invokeAction('calculate_projections', payload);
    return response;
  } catch (error) {
    console.error('[getProjectionsAction Error]', error);
    throw error;
  }
}
