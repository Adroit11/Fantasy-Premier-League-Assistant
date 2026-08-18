/**
 * Action: suggestLineupAction
 * Generates optimal Starting XI, Captain, Vice-Captain, and Bench sequence
 */

export async function suggestLineupAction(teamId, gameweek = null) {
  const numericId = Number(teamId);
  if (!numericId || isNaN(numericId)) {
    throw new Error('Valid team_id is required.');
  }

  const payload = { team_id: numericId };
  if (gameweek) {
    payload.gameweek = Number(gameweek);
  }

  try {
    const response = await window.fplAPI.invokeAction('suggest_lineup', payload);
    return response;
  } catch (error) {
    console.error('[suggestLineupAction Error]', error);
    throw error;
  }
}
