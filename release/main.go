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
	SubsurfaceStream subsurface_stream.ServerConfig `json:"stream"`
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

	ss, err := config.SubsurfaceStream.New()
	if err != nil {
		logrus.WithError(err).WithField("config", config.SubsurfaceStream).Panic("fail to create subsurface stream instance")
	}
	defer ss.Close()
	go ss.Run()
	logrus.Info("start subsurface stream")
	defer logrus.Info("exit subsurface stream")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc
}
