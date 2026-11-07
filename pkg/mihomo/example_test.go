package mihomo_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"mihomo-tui/pkg/mihomo"
)

// Example demonstrates basic usage of the Mihomo Client SDK.
func Example() {
	// Initialize client with functional options
	client, err := mihomo.New("http://127.0.0.1:9090",
		mihomo.WithSecret("your-secret"),
		mihomo.WithTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 1. Get version
	ver, err := client.GetVersion(ctx)
	if err != nil {
		log.Printf("GetVersion error: %v", err)
		return
	}
	fmt.Printf("Mihomo Core Version: %s\n", ver.Version)

	// 2. Switch mode
	if err := client.UpdateMode(ctx, "rule"); err != nil {
		log.Printf("UpdateMode error: %v", err)
		return
	}

	// 3. Test proxy node latency
	delay, err := client.TestDelay(ctx, "Node-01", "http://www.gstatic.com/generate_204", 3000)
	if err != nil {
		log.Printf("TestDelay error: %v", err)
		return
	}
	fmt.Printf("Latency: %s\n", mihomo.FormatDelay(delay))
}
