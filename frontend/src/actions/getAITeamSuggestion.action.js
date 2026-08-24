/**
 * Action: getAITeamSuggestionAction
 * Requests an AI-powered weekly team selection suggestion from the Go backend.
 * The backend calls Gemini 2.0 Flash and returns a markdown narrative + xP data.
 */

export async function getAITeamSuggestionAction(teamId, gameweek = null) {
  const numericId = Number(teamId);
  if (!numericId || isNaN(numericId)) {
    throw new Error('Valid team_id is required.');
  }

  const payload = { team_id: numericId };
  if (gameweek) {
    payload.gameweek = Number(gameweek);
  }

  try {
    const response = await window.fplAPI.invokeAction('ai/team-suggestion', payload);
    return response;
  } catch (error) {
    console.error('[getAITeamSuggestionAction Error]', error);
    throw error;
  }
}
