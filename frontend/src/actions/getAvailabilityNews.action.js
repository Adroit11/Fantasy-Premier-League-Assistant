/**
 * Action: getAvailabilityNewsAction
 * Fetches injury, suspension, and press conference status flags
 */

export async function getAvailabilityNewsAction(teamId = null, playerIds = null) {
  const payload = {};
  if (teamId) {
    payload.team_id = Number(teamId);
  }
  if (playerIds && Array.isArray(playerIds)) {
    payload.player_ids = playerIds.map(Number);
  }

  try {
    const response = await window.fplAPI.invokeAction('get_availability_news', payload);
    return response;
  } catch (error) {
    console.error('[getAvailabilityNewsAction Error]', error);
    throw error;
  }
}
