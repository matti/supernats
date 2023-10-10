package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

var stream chan *nats.Msg = make(chan *nats.Msg, 1)

func subscriber(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-stream:
			fmt.Println(msg.Subject)
			fmt.Println(string(msg.Data))
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	var subjectAll string
	var subjectQueue string
	var natsUrl string

	flag.StringVar(&subjectAll, "all", "", "")
	flag.StringVar(&subjectQueue, "queue", "", "")
	flag.StringVar(&natsUrl, "url", "", "")

	flag.Parse()

	if natsUrl == "" {
		log.Fatalln("url required")
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		subscriber(ctx)
		wg.Done()
	}()

	var nc *nats.Conn
	if n, err := nats.Connect(natsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(3),
		nats.ReconnectWait(3*time.Second),
		nats.Timeout(3*time.Second),
		nats.PingInterval(5*time.Second),
		nats.MaxPingsOutstanding(2),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("Disconnected, reason: %q\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("Reconnected to %v\n", nc.ConnectedUrl())
		}),
	); err != nil {
		log.Fatalln("err", err)
	} else {
		nc = n
	}

	for {
		if nc.IsConnected() {
			break
		}
		if !nc.IsReconnecting() {
			log.Fatalln("give up")
		}
		time.Sleep(time.Second)
	}

	log.Println("connected")

	if subjectAll != "" {
		if _, err := nc.ChanSubscribe(subjectAll, stream); err != nil {
			log.Fatalln(err)
		}
		log.Println("subscribed to all", subjectAll)
	}

	if subjectQueue != "" {
		if _, err := nc.ChanQueueSubscribe(subjectQueue, subjectQueue, stream); err != nil {
			log.Fatalln(err)
		}
		log.Println("subscribed to queue", subjectQueue)
	}

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			scanner.Scan()
			subject := scanner.Text()
			scanner.Scan()
			data := scanner.Text()

			m := &nats.Msg{
				Subject: subject,
				Data:    []byte(data),
			}
			if err := nc.PublishMsg(m); err != nil {
				log.Fatalln("publish error", err)
			}

			if err := nc.FlushTimeout(3 * time.Second); err != nil {
				log.Fatalln("flush failed", err)
			}
		}
	}()

	for {
		done := false
		select {
		case <-ctx.Done():
			done = true
			break
		default:
			if !nc.IsConnected() && !nc.IsReconnecting() {
				log.Println("give up")
				os.Exit(1)
			}

			time.Sleep(time.Second)
		}

		if done {
			break
		}
	}

	wg.Wait()
	log.Println("bye")
}
