package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"ctpcharts/config"
	"ctpcharts/handlers"
	"ctpcharts/middleware"
)

func main() {
	config.Load()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	engine := html.New("./templates", ".html")
	engine.Reload(true)
	engine.AddFuncMap(handlers.TemplateFuncs)

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use("/app/static", static.New("./static"))
	app.Use("/app/favicon.ico", static.New("./favicon.ico"))

	app.Get("/app/airports/departure", middleware.RequireAuth, handlers.DepartureAirportsPage)
	app.Get("/app/airports/arrival", middleware.RequireAuth, handlers.ArrivalAirportsPage)
	app.Get("/app/sectors", middleware.RequireAuth, handlers.SectorsPage)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("shutting down")
		app.Shutdown()
	}()

	log.Info().Str("port", config.C.Port).Msg("starting ctpcharts")
	if err := app.Listen(":" + config.C.Port); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
