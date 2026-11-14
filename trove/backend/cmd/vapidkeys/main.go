// Command vapidkeys generates a VAPID key pair for web push and prints it in
// .env form. Generate once: changing keys invalidates every subscription, so
// each device has to turn sale alerts on again.
package main

import (
	"fmt"
	"log"

	"github.com/SherClockHolmes/webpush-go"
)

func main() {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", public, private)
}
