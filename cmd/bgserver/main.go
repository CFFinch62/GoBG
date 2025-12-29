// Command bgserver runs the GoBG REST API server.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yourusername/bgengine/pkg/api"
	"github.com/yourusername/bgengine/pkg/engine"
)

const version = "0.1.0"

func main() {
	// Command line flags
	host := flag.String("host", "localhost", "Host to bind to (use 0.0.0.0 for all interfaces)")
	port := flag.Int("port", 8080, "Port to listen on")
	weightsFile := flag.String("weights", "data/gnubg.weights", "Path to neural network weights")
	bearoffFile := flag.String("bearoff", "data/gnubg_os0.bd", "Path to one-sided bearoff database")
	bearoffTSFile := flag.String("bearoff-ts", "data/gnubg_ts.bd", "Path to two-sided bearoff database")
	metFile := flag.String("met", "data/g11.xml", "Path to match equity table")
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	writeTimeout := flag.Duration("write-timeout", 30*time.Second, "HTTP write timeout")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("GoBG API Server v%s\n", version)
		os.Exit(0)
	}

	// Print startup banner
	log.Printf("GoBG API Server v%s", version)
	log.Printf("Loading engine data files...")

	// Create engine
	opts := engine.EngineOptions{
		WeightsFileText: *weightsFile,
		BearoffFile:     *bearoffFile,
		BearoffTSFile:   *bearoffTSFile,
		METFile:         *metFile,
	}

	eng, err := engine.NewEngine(opts)
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	log.Printf("Engine loaded successfully")

	// Create server config
	config := api.ServerConfig{
		Host:         *host,
		Port:         *port,
		ReadTimeout:  *readTimeout,
		WriteTimeout: *writeTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// Create and start server
	server := api.NewServer(eng, config, version)

	if err := server.ListenAndServeWithGracefulShutdown(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

