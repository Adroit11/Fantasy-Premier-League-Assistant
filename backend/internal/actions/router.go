package actions

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"fpl-assistant/internal/ai"
	"fpl-assistant/internal/config"
	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

type Router struct {
	App                          *fiber.App
	connectTeamAction            *ConnectTeamAction
	getTeamOverviewAction        *GetTeamOverviewAction
	getAvailabilityAction        *GetAvailabilityNewsAction
	calculateProjectionsAction   *CalculateProjectionsAction
	suggestLineupAction          *SuggestLineupAction
	suggestTransfersAction       *SuggestTransfersAction
	getAITeamSuggestionAction    *GetAITeamSuggestionAction
	getAIImprovementAction       *GetAIImprovementSuggestionAction
	recordGWScoreAction          *RecordGWScoreAction
	getGWScoreHistoryAction      *GetGWScoreHistoryAction
}

func NewRouter(client fpl.Client, engine scoring.Engine, optimizer scoring.Optimizer, database *db.Database, cfg *config.Config) *Router {
	app := fiber.New(fiber.Config{
		AppName:      "FPL Assistant Core (2026/2027)",
		ServerHeader: "FPL-Assistant-Core/1.0",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   "server_error",
				"message": err.Error(),
			})
		},
	})

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Requested-With",
	}))

	// Build AI client based on config
	var aiClient ai.AIClient
	if cfg.GeminiAPIKey != "" {
		aiClient = ai.NewGeminiClient(cfg.GeminiAPIKey, cfg.AIModel)
	} else {
		aiClient = ai.NewNoOpClient()
	}

	r := &Router{
		App:                        app,
		connectTeamAction:          NewConnectTeamAction(client, database),
		getTeamOverviewAction:      NewGetTeamOverviewAction(client),
		getAvailabilityAction:      NewGetAvailabilityNewsAction(client, database),
		calculateProjectionsAction: NewCalculateProjectionsAction(client, engine),
		suggestLineupAction:        NewSuggestLineupAction(client, engine, optimizer, database),
		suggestTransfersAction:     NewSuggestTransfersAction(client, engine),
		// AI-powered actions
		getAITeamSuggestionAction: NewGetAITeamSuggestionAction(client, engine, optimizer, database, aiClient, cfg.AIModel),
		getAIImprovementAction:    NewGetAIImprovementSuggestionAction(client, engine, database, aiClient, cfg.AIModel),
		recordGWScoreAction:       NewRecordGWScoreAction(client, engine, database),
		getGWScoreHistoryAction:   NewGetGWScoreHistoryAction(client, database),
	}

	r.routes()
	return r
}

func (r *Router) routes() {
	// Health check (Docker-compatible: /health and /healthz both work)
	healthHandler := func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"framework": "Fiber v2",
			"service":   "fpl-assistant-core",
			"version":   "1.0.0",
			"season":    "2026/2027",
		})
	}
	r.App.Get("/healthz", healthHandler)
	r.App.Get("/health", healthHandler)

	// Single Action API Group
	api := r.App.Group("/api/v1")

	// Existing endpoints
	api.Post("/team/connect", r.connectTeamAction.Handle)
	api.Post("/team/overview", r.getTeamOverviewAction.Handle)
	api.Post("/availability/news", r.getAvailabilityAction.Handle)
	api.Post("/projections/calculate", r.calculateProjectionsAction.Handle)
	api.Post("/lineup/suggest", r.suggestLineupAction.Handle)
	api.Post("/transfers/suggest", r.suggestTransfersAction.Handle)

	// AI-powered endpoints
	api.Post("/ai/team-suggestion", r.getAITeamSuggestionAction.Handle)
	api.Post("/ai/improvement", r.getAIImprovementAction.Handle)
	api.Post("/scores/record", r.recordGWScoreAction.Handle)
	api.Get("/scores/history/:team_id", r.getGWScoreHistoryAction.Handle)
}
