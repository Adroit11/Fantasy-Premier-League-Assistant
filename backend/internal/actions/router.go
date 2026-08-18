package actions

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

type Router struct {
	App                        *fiber.App
	connectTeamAction          *ConnectTeamAction
	getTeamOverviewAction      *GetTeamOverviewAction
	getAvailabilityAction      *GetAvailabilityNewsAction
	calculateProjectionsAction *CalculateProjectionsAction
	suggestLineupAction        *SuggestLineupAction
	suggestTransfersAction     *SuggestTransfersAction
}

func NewRouter(client fpl.Client, engine scoring.Engine, optimizer scoring.Optimizer, database *db.Database) *Router {
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

	r := &Router{
		App:                        app,
		connectTeamAction:          NewConnectTeamAction(client, database),
		getTeamOverviewAction:      NewGetTeamOverviewAction(client),
		getAvailabilityAction:      NewGetAvailabilityNewsAction(client, database),
		calculateProjectionsAction: NewCalculateProjectionsAction(client, engine),
		suggestLineupAction:        NewSuggestLineupAction(client, engine, optimizer, database),
		suggestTransfersAction:     NewSuggestTransfersAction(client, engine),
	}

	r.routes()
	return r
}

func (r *Router) routes() {
	// Health check
	r.App.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"framework": "Fiber v2",
			"service":   "fpl-assistant-core",
			"version":   "1.0.0",
			"season":    "2026/2027",
		})
	})

	// Single Action API Group
	api := r.App.Group("/api/v1")

	api.Post("/team/connect", r.connectTeamAction.Handle)
	api.Post("/team/overview", r.getTeamOverviewAction.Handle)
	api.Post("/availability/news", r.getAvailabilityAction.Handle)
	api.Post("/projections/calculate", r.calculateProjectionsAction.Handle)
	api.Post("/lineup/suggest", r.suggestLineupAction.Handle)
	api.Post("/transfers/suggest", r.suggestTransfersAction.Handle)
}
