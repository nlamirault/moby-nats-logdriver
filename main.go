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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/nlamirault/moby-nats-logdriver/driver"
	"github.com/nlamirault/moby-nats-logdriver/nats"
	"github.com/nlamirault/moby-nats-logdriver/version"
)

const (
	banner = "moby-nats-logdriver"
)

var (
	vrs bool
)

func init() {
	// parse flags
	flag.BoolVar(&vrs, "version", false, "print version and exit")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf("%s v%s\n", banner, version.Version))
		flag.PrintDefaults()
	}

	flag.Parse()

	if vrs {
		fmt.Printf("%s", version.Version)
		os.Exit(0)
	}
}

func main() {
	driver.SetLogLevel()
	logrus.WithField("version", version.Version).Info("Create the Nats log driver")
	h := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	natsClient, err := nats.NewClient()
	if err != nil {
		panic(err)
	}
	driver := driver.New(natsClient)
	driver.SetupHandlers(&h)
	if err := h.ServeUnix("moby-nats-logdriver", 0); err != nil {
		panic(err)
	}
	if err != natsClient.Disconnect(); err != nil {
		panic(err)
	}
}
