package logging

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

var once sync.Once

var log zerolog.Logger

func Get() *zerolog.Logger {
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano

		fileLogger := &lumberjack.Logger{
			Filename:   "rpi-provisioner.log",
			MaxSize:    5, //
			MaxBackups: 10,
			MaxAge:     14,
			Compress:   true,
		}

		output := zerolog.MultiLevelWriter(fileLogger)

		log = zerolog.New(output).
			Level(zerolog.DebugLevel).
			With().
			Timestamp().
			Logger()
	})

	return &log
}
