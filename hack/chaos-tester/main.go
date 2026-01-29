package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	toxiproxyClient "github.com/Shopify/toxiproxy/v2/client"
	"k8s.io/apimachinery/pkg/util/wait"
)

func timestamp() string {
	return time.Now().Format(time.ANSIC) + "> "
}

func println(str ...any) {
	fmt.Printf("%s", timestamp())
	fmt.Println(str...)
}

func printf(format string, str ...any) {
	fmt.Printf("%s", timestamp())
	fmt.Printf(format, str...)
}

func main() {

	toxiproxyServerAddress := "127.0.0.1:8474"
	proxyListenAddress := "127.0.0.1:8475"
	upstreamAddress := "127.0.0.1:8443"

	println("Waiting for toxiproxy server")

	// Wait for the Toxiproxy server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		conn, err := net.Dial("tcp", "localhost:8474")
		if err == nil {
			conn.Close()
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		printf("Toxiproxy server not ready: %v\n", err)
		return
	}

	println("Creating proxy")
	client := toxiproxyClient.NewClient(toxiproxyServerAddress)
	proxy, err := client.CreateProxy("test", proxyListenAddress, upstreamAddress)
	if err != nil {
		println("Error:", err)
		return
	}

	println("Waiting")

	randomEnableDisable := true

	for {

		if randomEnableDisable {
			// Generate a random duration between X and Y seconds for the proxy to stay enabled
			enabledDuration := time.Duration(rand.Intn(10)+5) * time.Second
			printf("Proxy enabled, waiting %v before disabling...\n", enabledDuration)
			time.Sleep(enabledDuration)

			// Disable the proxy to simulate a network failure
			if err := proxy.Disable(); err != nil {
				printf("Error disabling proxy: %v\n", err)
				return
			}
			println("Proxy disabled")

			// Generate a random duration between X and Y seconds for the proxy to stay disabled
			disabledDuration := time.Duration(rand.Intn(5)+5) * time.Second
			printf("Proxy disabled, waiting %v before re-enabling...\n", disabledDuration)
			time.Sleep(disabledDuration)

			// Re-enable the proxy to restore connectivity
			if err := proxy.Enable(); err != nil {
				printf("Error enabling proxy: %v\n", err)
				return
			}
			println("Proxy enabled")
		} else {
			time.Sleep(1 * time.Second)
		}

	}

}
