/**
 * Action: connectTeamAction
 * Single Action File for connecting manager team_id
 */

export async function connectTeamAction(teamId) {
  const numericId = Number(teamId);
  if (!numericId || isNaN(numericId) || numericId <= 0) {
    throw new Error('Please enter a valid positive numeric FPL Team ID.');
  }

  try {
    const response = await window.fplAPI.invokeAction('connect_team', {
      team_id: numericId,
    });
    return response;
  } catch (error) {
    console.error('[connectTeamAction Error]', error);
    throw error;
  }
}
