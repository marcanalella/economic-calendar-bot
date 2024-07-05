package main

import (
	"bot/conf"
	"bot/internal"
	"context"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"log"
	"net/http"
	"os"
)

func main() {

	cfg, err := conf.Load()
	if err != nil {
		log.Fatalf("could not decode config %s\n", err.Error())
	}

	recipients, err := conf.LoadRecipients()
	if err != nil {
		log.Fatalf("could not decode recipients %s\n", err.Error())
	}

	var sheetsService *sheets.Service

	creeds, err := os.ReadFile(cfg.KeyFile)
	if err != nil {
		log.Fatalf("Unable to read credentials file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(creeds, sheets.SpreadsheetsScope)
	if err != nil {
		log.Fatalf("Unable to create JWT config: %v", err)
	}

	client := config.Client(context.Background())
	sheetsService, err = sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Google Sheets service: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}

	server := &http.Server{
		Addr:    cfg.Address + ":" + port,
		Handler: buildHandler(cfg),
	}

	scheduler := internal.NewService(cfg)
	scheduler.Readyz(recipients)
	//CALENDAR NEWS SCHEDULER
	//TODO API NOT VALID ANYMORE - FIND ANOTHER FREE SERVICE
	//scheduler.ScheduledNewsNotification(recipients)

	//XAU SCHEDULER
	scheduler.ScheduledXauNotification(recipients, cfg.SpreadsheetId, cfg.ReadRange, sheetsService)
	scheduler.ScheduledXauSheetUpdate(recipients, cfg.SpreadsheetId, cfg.WriteRange, cfg.SheetId, cfg.FinancialModelingPrepUrl, sheetsService)

	select {}

	log.Println("Listening ", server.Addr)
	err = server.ListenAndServe()
	log.Fatalln(err)
}

func buildHandler(cfg conf.Config) http.Handler {

	//all APIs are under "/api/v1" path prefix
	router := mux.NewRouter()
	router.Use(loggingMiddleware)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	})

	routerGroup := router.PathPrefix("/api/v1").Subrouter()
	internal.RegisterHandlers(routerGroup, internal.NewService(cfg))
	handler := c.Handler(router)
	return handler
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
