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

var nc *nats.Conn

var flushNeeded bool
var outbound chan *nats.Msg = make(chan *nats.Msg, 32)
var inbound chan *nats.Msg = make(chan *nats.Msg, 32)

type input struct {
	command string
	args    string
	data    string
}

var inputs chan *input = make(chan *input)
var disconnected chan bool = make(chan bool)

func stdinReader() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		command := scanner.Text()
		scanner.Scan()
		args := scanner.Text()
		scanner.Scan()
		data := scanner.Text()

		inputs <- &input{
			command: command,
			args:    args,
			data:    data,
		}
	}
}

func disconnectMonitor() {
	for {
		if nc.IsConnected() || nc.IsReconnecting() {
			time.Sleep(time.Second)
		} else {
			disconnected <- true
			return
		}
	}
}

func flusher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if flushNeeded {
				if err := nc.FlushTimeout(5 * time.Second); err != nil {
					log.Println("flush failed", err)
				}

				flushNeeded = false
			}
			time.Sleep(1 * time.Second)
		}
	}
}
func subscriber(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-outbound:
			if err := nc.PublishMsg(msg); err != nil {
				log.Fatalln("publish error", err)
			}
			log.Println("published to", msg.Subject)
			flushNeeded = true
		case msg := <-inbound:
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

	flag.Parse()

	natsUrl := flag.Arg(0)
	if natsUrl == "" {
		log.Fatalln("url required")
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		subscriber(ctx)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		flusher(ctx)
		wg.Done()
	}()

	if n, err := nats.Connect(natsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(5*time.Second),
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

	done := false
	for {
		select {
		case <-ctx.Done():
			done = true
		default:
			if nc.IsConnected() {
				done = true
				break
			}
			if !nc.IsReconnecting() {
				log.Fatalln("give up")
			}
			log.Println("waiting for connection")
			time.Sleep(time.Second)
		}

		if done {
			break
		}
	}

	if nc.IsConnected() {
		log.Println("connected")
	} else {
		os.Exit(1)
	}

	go disconnectMonitor()
	go stdinReader()

	subscriptionsQueue := make(map[string]*nats.Subscription)
	subscriptionsBroadcast := make(map[string]*nats.Subscription)
	for {
		done := false
		select {
		case <-ctx.Done():
			done = true
			break
		case <-disconnected:
			log.Println("give up")
			os.Exit(1)
		case i := <-inputs:
			switch i.command {
			case "quit", "exit":
				done = true
				break
			case "publish":
				subject := i.args
				outbound <- &nats.Msg{
					Subject: subject,
					Data:    []byte(i.data),
				}
			case "unsubscribe":
				mode := i.args
				subject := i.data

				var subscriptions map[string]*nats.Subscription
				switch mode {
				case "queue":
					subscriptions = subscriptionsQueue
				case "broadcast":
					subscriptions = subscriptionsBroadcast
				default:
					log.Fatal("unknown mode", mode)
				}

				if subscriptions[subject] == nil {
					log.Fatalln("no subscription with subject", subject)
				} else if err := subscriptions[subject].Unsubscribe(); err != nil {
					log.Fatalln(err)
				} else {
					subscriptions[subject] = nil
					log.Println("unsubscribed from", subject)
				}

			case "subscribe":
				mode := i.args
				subject := i.data
				switch mode {
				case "queue":
					if s, err := nc.ChanQueueSubscribe(subject, subject, inbound); err != nil {
						log.Fatalln(err)
					} else {
						subscriptionsQueue[subject] = s
					}
					log.Println("subscribed as queue to", subject)
				case "broadcast":
					if s, err := nc.ChanSubscribe(subject, inbound); err != nil {
						log.Fatalln(err)
					} else {
						subscriptionsBroadcast[subject] = s
					}
					log.Println("subscribed as broadcast to", subject)
				default:
					log.Fatalln("unknown mode", mode)
				}
			default:
				log.Fatalln("unknown command", i.command)
			}

		}
		if done {
			break
		}
	}
	cancel()
	wg.Wait()
	log.Println("bye")
}
