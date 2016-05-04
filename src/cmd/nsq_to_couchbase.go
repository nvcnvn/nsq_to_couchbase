package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/couchbase/gocb"
	"github.com/nsqio/go-nsq"
	"github.com/satori/go.uuid"
)

type StringArray []string

func (a *StringArray) Set(s string) error {
	*a = append(*a, s)
	return nil
}

func (a *StringArray) String() string {
	return strings.Join(*a, ",")
}

var (
	connStr     = flag.String("connStr", "", "Couchbase connection string")
	bucketStr   = flag.String("bucket", "", "Couchbase bucket")
	bucketPwd   = flag.String("bucketPwd", "", "Couchbase bucket password (optional)")
	topic       = flag.String("topic", "", "NSQ topic")
	channel     = flag.String("channel", "", "NSQ channel")
	maxInFlight = flag.Int("max-in-flight", 200, "max number of messages to allow in flight")
	jsonEncoded = flag.Bool("json", true, "determine if message encoded with JSON")

	nsqdTCPAddrs     = StringArray{}
	lookupdHTTPAddrs = StringArray{}
	keyFields        = StringArray{}
)

func init() {
	flag.Var(&nsqdTCPAddrs, "nsqd-tcp-address", "nsqd TCP address (may be given multiple times)")
	flag.Var(&lookupdHTTPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
	flag.Var(&keyFields, "key-fields", "if message encoded with JSON, use this list to lookup the key field")
}

type CouchbaseHandler struct {
	Bucket    *gocb.Bucket
	KeyFields StringArray
}

func (ch *CouchbaseHandler) insertMessage(key string, m *nsq.Message) error {
	_, err := ch.Bucket.Insert(key, m.Body, 0)
	if err != nil {
		log.Println("error when insert", err)
		m.Requeue(-1)
		return err
	}

	// success finish
	m.Finish()
	return nil
}

func (ch *CouchbaseHandler) HandleMessage(m *nsq.Message) error {
	m.DisableAutoResponse()

	if *jsonEncoded {
		if len(ch.KeyFields) == 0 {
			v := parseObject{}
			if err := json.Unmarshal(m.Body, &v); err != nil {
				m.Finish()
				return err
			}
			if len(v.MessageID) > 0 {
				return ch.insertMessage(v.MessageID, m)
			}
		}
		v := map[string]interface{}{}
		if err := json.Unmarshal(m.Body, &v); err != nil {
			m.Finish()
			return err
		}
		for _, field := range ch.KeyFields {
			key, ok := v[field]
			if keyStr, isStr := key.(string); ok && isStr {
				return ch.insertMessage(keyStr, m)
			} else {
				m.Finish()
				return errors.New("nsq_to_couchbase: key field must be string")
			}
		}
	}

	// so, not a jsonEncoded or no key field match
	// then just generate an uuid for them
	return ch.insertMessage(base64.RawStdEncoding.EncodeToString(uuid.NewV4().Bytes()), m)
}

func main() {
	cfg := nsq.NewConfig()
	flag.Var(&nsq.ConfigFlag{cfg}, "consumer-opt", "option to passthrough to nsq.Consumer (may be given multiple times, http://godoc.org/github.com/nsqio/go-nsq#Config)")

	flag.Parse()

	if *channel == "" {
		rand.Seed(time.Now().UnixNano())
		*channel = fmt.Sprintf("couchbase%06d#ephemeral", rand.Int()%999999)
	}

	if *topic == "" {
		log.Fatal("--topic is required")
	}

	if *connStr == "" {
		log.Fatal("--connStr is required")
	}

	if *bucketStr == "" {
		log.Fatal("--bucket is required")
	}

	if len(nsqdTCPAddrs) == 0 && len(lookupdHTTPAddrs) == 0 {
		log.Fatal("--nsqd-tcp-address or --lookupd-http-address required")
	}
	if len(nsqdTCPAddrs) > 0 && len(lookupdHTTPAddrs) > 0 {
		log.Fatal("use --nsqd-tcp-address or --lookupd-http-address not both")
	}

	cluster, err := gocb.Connect(*connStr)
	if err != nil {
		log.Fatalf("Cannot connect to Couchbase (connStr: %s)", *connStr)
	}

	bucket, err := cluster.OpenBucket(*bucketStr, *bucketPwd)
	if err != nil {
		log.Fatalf("Cannot connect to Couchbase bucket %s (password: '%s')", *bucketStr, *bucketPwd)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg.UserAgent = fmt.Sprintf("nsq_to_couchbase go-nsq/%s", nsq.VERSION)
	cfg.MaxInFlight = *maxInFlight

	consumer, err := nsq.NewConsumer(*topic, *channel, cfg)
	if err != nil {
		log.Fatal(err)
	}

	consumer.AddHandler(&CouchbaseHandler{
		Bucket:    bucket,
		KeyFields: keyFields,
	})

	err = consumer.ConnectToNSQDs(nsqdTCPAddrs)
	if err != nil {
		log.Fatal(err)
	}

	err = consumer.ConnectToNSQLookupds(lookupdHTTPAddrs)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
		}
	}
}
