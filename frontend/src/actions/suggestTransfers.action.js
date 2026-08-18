/**
 * Action: suggestTransfersAction
 * Generates prioritized transfer upgrade recommendations with net xP delta
 */

export async function suggestTransfersAction(teamId, maxTransfers = 5, gameweek = null) {
  const numericId = Number(teamId);
  if (!numericId || isNaN(numericId)) {
    throw new Error('Valid team_id is required.');
  }

  const payload = {
    team_id: numericId,
    max_transfers: Number(maxTransfers),
  };
  if (gameweek) {
    payload.gameweek = Number(gameweek);
  }

  try {
    const response = await window.fplAPI.invokeAction('suggest_transfers', payload);
    return response;
  } catch (error) {
    console.error('[suggestTransfersAction Error]', error);
    throw error;
  }
}
