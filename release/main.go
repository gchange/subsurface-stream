package main

import (
	"encoding/json"
	"flag"
	"github.com/gchange/subsurface-stream"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

type LogrusConfig struct {
}

type Config struct {
	LogrusConfig LogrusConfig `json:"logger"`
	SubsurfaceStream []subsurface_stream.Config `json:"subsurface"`
}

func (config *LogrusConfig) Init() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	return nil
}

func main() {
	flag.Parse()
	fileName := flag.String("config", "config.json", "json config file")
	buf, err := ioutil.ReadFile(*fileName)
	if err != nil {
		logrus.WithError(err).WithField("file", fileName).Panic("fail to read config file")
	}

	var config Config
	err = json.Unmarshal(buf, &config)
	if err != nil {
		logrus.WithError(err).WithField("config", string(buf)).Panic("fail to unmarshal config")
	}

	err = config.LogrusConfig.Init()
	if err != nil {
		logrus.WithError(err).WithField("config", config.LogrusConfig).Panic("fail to init logrus")
	}

	streams := make([]*subsurface_stream.SubsurfaceStream, 0)
	defer func() {
		for _, ss := range streams {
			ss.Close()
		}
	}()
	for _, c := range config.SubsurfaceStream {
		ss, err := c.New()
		if err != nil {
			logrus.WithError(err).WithField("config", c).Panic("fail to init stream")
		}
		streams = append(streams, ss)
		go ss.Run()
		logrus.WithField("config", c).Debug("run stream")
	}

	logrus.Info("start subsurface stream")
	defer logrus.Info("exit subsurface stream")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc
}
