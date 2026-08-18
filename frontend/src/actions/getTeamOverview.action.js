/**
 * Action: getTeamOverviewAction
 * Fetches manager squad breakdown and player metrics
 */

export async function getTeamOverviewAction(teamId, gameweek = null) {
  const numericId = Number(teamId);
  if (!numericId || isNaN(numericId)) {
    throw new Error('Valid team_id is required.');
  }

  const payload = { team_id: numericId };
  if (gameweek) {
    payload.gameweek = Number(gameweek);
  }

  try {
    const response = await window.fplAPI.invokeAction('get_team_overview', payload);
    return response;
  } catch (error) {
    console.error('[getTeamOverviewAction Error]', error);
    throw error;
  }
}
