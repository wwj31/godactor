package test

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestJetStreamWorkQueue(t *testing.T) {
	// Use the env variable if running in the container, otherwise use the default.
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}

	// Create an unauthenticated connection to NATS.
	nc, _ := nats.Connect(url)
	defer nc.Drain()

	// Access `JetStreamContext` to use the JS APIs.
	js, _ := nc.JetStream()

	// ### Creating the stream
	// Define the stream configuration, specifying `WorkQueuePolicy` for
	// retention, and create the stream.
	cfg := &nats.StreamConfig{
		Name:      "EVENTS",
		Retention: nats.WorkQueuePolicy,
		Subjects:  []string{"events.>"},
	}

	if _, err := js.AddStream(cfg); err != nil {
		if jsErr, ok := err.(nats.JetStreamError); !ok || jsErr.APIError().ErrorCode != nats.JSErrCodeStreamNameInUse {
			assert.NoError(t, err)
		}
	}

	fmt.Println("created the stream")

	// ### Queue messages
	// Publish a few messages.
	js.Publish("events.us.page_loaded", []byte("msg1"))
	js.Publish("events.eu.mouse_clicked", []byte("msg2"))
	js.Publish("events.us.input_focused", []byte("msg3"))
	js.Publish("events.us.input_focused", []byte("msg4"))
	fmt.Println("published 4 messages")

	// Checking the stream info, we see three messages have been queued.
	fmt.Println("# Stream info without any consumers")
	printStreamState(js, cfg.Name)

	// ### Adding a consumer
	// Now let's add a consumer and publish a few more messages. It can be
	// a [push][push] or [pull][pull] consumer. For this example, we are
	// defining a pull consumer.
	// [push]: /examples/jetstream/push-consumer/go
	// [pull]: /examples/jetstream/pull-consumer/go
	sub1, _ := js.PullSubscribe("", "processor-1", nats.BindStream(cfg.Name))

	// Fetch and ack the queued messages.
	t1 := time.Now()
	msgs, _ := sub1.Fetch(100, nats.MaxWait(10*time.Second))
	for _, msg := range msgs {
		fmt.Println(string(msg.Data))
		msg.AckSync()
	}
	fmt.Println("delay ", time.Now().Sub(t1).String())

	// Checking the stream info again, we will notice no messages
	// are available.
	fmt.Println("\n# Stream info with one consumer")
	printStreamState(js, cfg.Name)

	// ### Exclusive non-filtered consumer
	// As noted in the description above, work-queue streams can only have
	// at most one consumer with interest on a subject at any given time.
	// Since the pull consumer above is not filtered, if we try to create
	// another one, it will fail.
	_, err := js.PullSubscribe("", "processor-2", nats.BindStream(cfg.Name))
	fmt.Println("\n# Create an overlapping consumer")
	fmt.Println(err)

	// However if we delete the first one, we can then add the new one.
	sub1.Unsubscribe()

	sub2, err := js.PullSubscribe("", "processor-2", nats.BindStream(cfg.Name))
	fmt.Printf("created the new consumer? %v\n", err == nil)
	sub2.Unsubscribe()

	// ### Multiple filtered consumers
	// To create multiple consumers, a subject filter needs to be applied.
	// For this example, we could scope each consumer to the geo that the
	// event was published from, in this case `us` or `eu`.
	fmt.Println("\n# Create non-overlapping consumers")
	sub1, _ = js.PullSubscribe("events.us.>", "processor-us", nats.BindStream(cfg.Name))
	sub2, _ = js.PullSubscribe("events.eu.>", "processor-eu", nats.BindStream(cfg.Name))

	js.Publish("events.eu.mouse_clicked", nil)
	js.Publish("events.us.page_loaded", nil)
	js.Publish("events.us.input_focused", nil)
	js.Publish("events.eu.page_loaded", nil)
	fmt.Println("published 4 messages")

	msgs, _ = sub1.Fetch(2)
	for _, msg := range msgs {
		fmt.Printf("us sub got: %s\n", msg.Subject)
		msg.Ack()
	}

	msgs, _ = sub2.Fetch(2)
	for _, msg := range msgs {
		fmt.Printf("eu sub got: %s\n", msg.Subject)
		msg.Ack()
	}
}
