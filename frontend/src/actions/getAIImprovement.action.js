/**
 * Action: getAIImprovementAction
 * Requests an AI-powered post-GW improvement suggestion.
 * Reviews the previous GW result and generates transfer priorities + chip advice.
 */

export async function getAIImprovementAction(teamId, gameweek = null) {
  const numericId = Number(teamId);
  if (!numericId || isNaN(numericId)) {
    throw new Error('Valid team_id is required.');
  }

  const payload = { team_id: numericId };
  if (gameweek) {
    payload.gameweek = Number(gameweek);
  }

  try {
    const response = await window.fplAPI.invokeAction('ai/improvement', payload);
    return response;
  } catch (error) {
    console.error('[getAIImprovementAction Error]', error);
    throw error;
  }
}
