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

package nats

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	gonats "github.com/nats-io/nats"
)

const (
	// NatsServers URI of the Nats servers
	natsServers = "NATS_ADDRESS"

	// NatsSubject define which subject to used
	natsSubject = "NATS_SUBJECT"

	driverName = "nats-log-driver"
)

var (
	ErrNatsUnknownServers = errors.New("Nats servers cannot be empty")
	ErrNatsSubject        = errors.New("Nats subject cannot be empty")
)

// An mapped version of logger.Message where Line is a String, not a byte array
type LogMessage struct {
	Message          string            `json:"message"`
	ContainerId      string            `json:"container_id"`
	ContainerName    string            `json:"container_name"`
	ContainerCreated time.Time         `json:"container_created"`
	ImageId          string            `json:"image_id"`
	ImageName        string            `json:"image_name"`
	Command          string            `json:"command"`
	Hostname         string            `json:"hostname"`
	Tag              string            `json:"tag"`
	Extra            map[string]string `json:"extra"`
}

type Client struct {
	nc      *gonats.Conn
	conn    *gonats.EncodedConn
	subject string
	address string
	// fields  map[string]interface{}
}

func NewClient() (*Client, error) {
	addr := os.Getenv(natsServers)
	if addr == "" {
		return nil, ErrNatsUnknownServers
	}

	subject := os.Getenv(natsSubject)
	if subject == "" {
		return nil, ErrNatsSubject
	}

	opts := gonats.Options{
		Url:            addr,
		Secure:         true,
		AllowReconnect: true,
		MaxReconnect:   10,
		ReconnectWait:  5 * time.Second,
		Timeout:        1 * time.Second,
	}
	nc, err := opts.Connect()
	if err != nil {
		return nil, err
	}

	connEncoded, err := gonats.NewEncodedConn(nc, gonats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	// Set handlers to log in case of events related to the established connection
	nc.SetDisconnectHandler(func(c *gonats.Conn) {
		logrus.WithField("driver", driverName).Warnf("nats: disconnected")
	})

	nc.SetReconnectHandler(func(c *gonats.Conn) {
		logrus.WithField("driver", driverName).Warnf("nats: reconnected to %q", c.ConnectedUrl())
	})

	nc.SetClosedHandler(func(c *gonats.Conn) {
		logrus.WithField("driver", driverName).Warnf("nats: connection closed")
	})

	logrus.WithField("driver", driverName).Infof("nats: connected to %q, status: %d", nc.ConnectedUrl(), connEncoded.Conn.Status())
	return &Client{
		address: addr,
		subject: subject,
		conn:    connEncoded,
		nc:      nc,
	}, nil
}

func (client *Client) Disconnect() error {
	logrus.WithField("driver", driverName).Infof("Nats broker disconnecting")
	if client.conn != nil {
		client.conn.Close()
	}
	return nil
}

func (client *Client) LogToNats(logMessage LogMessage) error {
	logrus.WithField("driver", driverName).Debugf("Send to nats: %s", logMessage)
	msg, err := json.Marshal(logMessage)
	if err != nil {
		return err
	}
	return client.conn.Publish(client.subject, msg)
}
