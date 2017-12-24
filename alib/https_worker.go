package alib

import (
	"bytes"
	"crypto/tls"
	"encoding/pem"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gocraft/work"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var redisPool = &redis.Pool{
	MaxActive: 5,
	MaxIdle:   5,
	Wait:      true,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", "172.29.152.196:6379")
	},
}

type Context struct {
	customerID int64
}

func StartWorker(conncurrency uint) {
	// Make a new pool. Arguments:
	// Context{} is a struct that will be the context for the request.
	// 10 is the max concurrency
	// "my_app_namespace" is the Redis namespace
	// redisPool is a Redis pool
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("The worker is running. Waiting for request.")

	pool := work.NewWorkerPool(Context{}, conncurrency, "cert_app", redisPool)

	// Add middleware that will be executed for each job
	pool.Middleware((*Context).Log)
	pool.Middleware((*Context).FindCustomer)

	// Map the name of jobs to handler functions
	pool.Job("get_certificate", (*Context).GetCertificatesPEM)

	// Customize options:
	pool.JobWithOptions("export", work.JobOptions{Priority: 10, MaxFails: 1}, (*Context).Export)

	// Start processing jobs
	pool.Start()

	// Wait for a signal to quit:
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	<-signalChan

	// Stop the pool
	pool.Stop()
}

func (c *Context) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	//fmt.Println("Starting job: ", job.Name)
	return next()
}

func (c *Context) FindCustomer(job *work.Job, next work.NextMiddlewareFunc) error {
	// If there's a customer_id param, set it in the context for future middleware and handlers to use.
	if _, ok := job.Args["customer_id"]; ok {
		c.customerID = job.ArgInt64("customer_id")
		if err := job.ArgError(); err != nil {
			return err
		}
	}

	return next()
}

func (c *Context) GetCertificatesPEM(job *work.Job) error {
	// Extract arguments:
	addr := job.ArgString("address")
	if err := job.ArgError(); err != nil {
		return err
	}

	_, err := GetCertificatesPEMFrom(addr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get Certificate.")
	}

	return nil
}

func (c *Context) Export(job *work.Job) error {
	return nil
}

func GetCertificatesPEMFrom(address string) (string, error) {
	log.Print("Start....")
	timeout := 5
	conn, err := net.DialTimeout("tcp", address, time.Duration(timeout)*time.Second)
	if err != nil {
		log.Error().Err(err).Str("tls", address).Msg("Failed to establish the tcp connection")
		return "", err
	}
	connn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})
	err = connn.Handshake()
	if err != nil {
		log.Error().Err(err).Str("tls", address).Msg("Failed to Handshake with tls connection")
		return "", err
	}
	defer connn.Close()
	var b bytes.Buffer
	for _, cert := range connn.ConnectionState().PeerCertificates {
		err := pem.Encode(&b, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		if err != nil {
			log.Error().Err(err).Str("tls", address).Msg("Failed to encode certificates from the tls connection")
			return "", err
		}
	}
	log.Info().Int("pid", os.Getpid()).Msg("Get Certificates successfully!")
	return b.String(), nil
}
