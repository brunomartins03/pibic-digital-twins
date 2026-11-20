// Command relay bridges the robotic arm (MQTT) and Decentraland scenes
// (WebSocket). Configuration is read from environment variables; see the
// module documentation in docs/.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pibic-digital-twins/relay-server/internal/relay"
)

func main() {
	addr := envOr("RELAY_ADDR", ":8080")
	sceneToken := os.Getenv("SCENE_TOKEN")
	mqttCfg := relay.MQTTConfig{
		BrokerURL:   envOr("MQTT_BROKER", "tcp://localhost:1883"),
		ClientID:    envOr("MQTT_CLIENT_ID", "robotarm-relay"),
		Username:    os.Getenv("MQTT_USERNAME"),
		Password:    os.Getenv("MQTT_PASSWORD"),
		TopicPrefix: envOr("MQTT_TOPIC_PREFIX", "robotarm"),
	}
	if sceneToken == "" {
		log.Println("WARNING: SCENE_TOKEN not set, /ws/scene accepts unauthenticated clients")
	}

	hub := relay.NewHub()
	link := relay.ConnectMQTT(mqttCfg, hub)
	defer link.Close()

	srv := &http.Server{
		Addr:              addr,
		Handler:           relay.NewServer(hub, link, sceneToken).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("http: listening on %s (topic prefix %q)", addr, mqttCfg.TopicPrefix)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http: shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
