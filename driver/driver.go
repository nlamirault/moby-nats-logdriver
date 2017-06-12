// Copyright (C) 2017 Nicolas Lamirault <nicolas.lamirault@gmail.com>

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/containerd/fifo"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/daemon/logger/loggerutils"
	protoio "github.com/gogo/protobuf/io"
	"github.com/pkg/errors"

	"github.com/nlamirault/moby-nats-logdriver/nats"
)

type logPair struct {
	logMessage *nats.LogMessage
	stream     io.ReadCloser
	info       logger.Info
}

type driver struct {
	mu         sync.Mutex
	logs       map[string]*logPair
	idx        map[string]*logPair
	logger     logger.Logger
	natsClient *nats.Client
}

func New(natsClient *nats.Client) *driver {
	return &driver{
		logs:       make(map[string]*logPair),
		idx:        make(map[string]*logPair),
		natsClient: natsClient,
	}
}

func (d *driver) StartLogging(file string, logCtx logger.Info) error {
	logrus.WithField("id", logCtx.ContainerID).WithField("file", file).WithField("logpath", logCtx.LogPath).Infof("Start logging")
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("logger for %q already exists", file)
	}
	d.mu.Unlock()

	if logCtx.LogPath == "" {
		logCtx.LogPath = filepath.Join("/var/log/docker", logCtx.ContainerID)
	}
	if err := os.MkdirAll(filepath.Dir(logCtx.LogPath), 0755); err != nil {
		return errors.Wrap(err, "error setting up logger dir")
	}

	stream, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening logger fifo: %q", file)
	}

	logPair := &logPair{
		info:   logCtx,
		stream: stream,
	}

	d.mu.Lock()
	d.logs[file] = logPair
	d.mu.Unlock()

	go consumeLog(d.natsClient, logPair)

	return nil
}

func (d *driver) StopLogging(file string) error {
	logrus.WithField("file", file).Infof("Stop logging")
	d.mu.Lock()
	lf, ok := d.logs[file]
	if ok {
		lf.stream.Close()
		delete(d.logs, file)
	}
	d.mu.Unlock()
	return nil
}

func (d *driver) ReadLogs(info logger.Info, config logger.ReadConfig) (io.ReadCloser, error) {
	logrus.WithField("info", info).Debugf("Read logs")
	return nil, nil
}

func consumeLog(natsClient *nats.Client, lf *logPair) {
	dec := protoio.NewUint32DelimitedReader(lf.stream, binary.BigEndian, 1e6)
	defer dec.Close()
	var buf logdriver.LogEntry
	for {
		if err := dec.ReadMsg(&buf); err != nil {
			if err == io.EOF {
				logrus.WithField("id", lf.info.ContainerID).WithError(err).Debug("shutting down log logger")
				lf.stream.Close()
				return
			}
			dec = protoio.NewUint32DelimitedReader(lf.stream, binary.BigEndian, 1e6)
		}

		var logMessage nats.LogMessage
		logMessage.Message = string(buf.Line)
		logMessage.ContainerId = lf.info.ContainerID
		logMessage.ContainerName = lf.info.ContainerName
		logMessage.ContainerCreated = lf.info.ContainerCreated
		logMessage.ImageName = lf.info.ContainerImageName
		logMessage.ImageId = lf.info.ContainerImageID

		tag, err := loggerutils.ParseLogTag(lf.info, loggerutils.DefaultTemplate)
		if err != nil {
			logrus.WithField("id", lf.info.ContainerID).WithError(err).WithField("message", logMessage).Error("error extract log tag informations")
		}
		logMessage.Tag = tag

		extra, err := lf.info.ExtraAttributes(nil)
		if err != nil {
			logrus.WithField("id", lf.info.ContainerID).WithError(err).WithField("message", logMessage).Error("error extract log extra informations")
		}
		logMessage.Extra = extra

		hostname, err := lf.info.Hostname()
		if err != nil {
			logrus.WithField("id", lf.info.ContainerID).WithError(err).WithField("message", logMessage).Error("error extract log hostname informations")
		}
		logMessage.Hostname = hostname

		if err := natsClient.LogToNats(logMessage); err != nil {
			logrus.WithField("id", lf.info.ContainerID).WithError(err).WithField("message", logMessage).Error("error writing log message")
			continue
		}

		buf.Reset()
	}
}
