package main

import (
	"bot/conf"
	"bot/internal"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"net/http"
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

	server := &http.Server{
		Addr:    cfg.Address + ":" + cfg.Port,
		Handler: buildHandler(cfg),
	}

	scheduler := internal.NewService(cfg)
	scheduler.ScheduledNotification(recipients)
	scheduler.Readyz(recipients)
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
		AllowedHeaders:   []string{"Authorization,Content-Type"},
		AllowedMethods:   []string{"GET,POST,PUT,DELETE,PATCH,OPTIONS"},
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
