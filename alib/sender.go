package alib

import (
	"os"

	"github.com/garyburd/redigo/redis"
	"github.com/gocraft/work"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Make a redis pool
var redisPool1 = &redis.Pool{
	MaxActive: 5,
	MaxIdle:   5,
	Wait:      true,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	},
}

// Make an enqueuer with a particular namespace
var enqueuer = work.NewEnqueuer("cert_app", redisPool1)

func SendRequest(target string, num int) {
	// Enqueue a job named "get_certificate" with the specified parameters.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	for i := 0; i < num; i++ {
		job, err := enqueuer.Enqueue("get_certificate", work.Q{"address": target, "customer_id": 4})
		log.Print(job)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get certificate.")
		}

	}
}
