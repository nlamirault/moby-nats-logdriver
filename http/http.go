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

package http

import (
	"encoding/json"
	"errors"
	"io"
	gohttp "net/http"

	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/nlamirault/moby-nats-logdriver/driver"
)

type StartLoggingRequest struct {
	File string
	Info logger.Info
}

type StopLoggingRequest struct {
	File string
}

type CapabilitiesResponse struct {
	Err string
	Cap logger.Capability
}

type ReadLogsRequest struct {
	Info   logger.Info
	Config logger.ReadConfig
}

type response struct {
	Err string
}

func SetupHandlers(h *sdk.Handler, d *driver.Driver) {
	h.HandleFunc("/LogDriver.StartLogging", startLoggingHandler(d))
	h.HandleFunc("/LogDriver.StopLogging", stopLoggingHandler(d))
	h.HandleFunc("/LogDriver.Capabilities", capabilitiesHandler(d))
	h.HandleFunc("/LogDriver.ReadLogs", readLogsHandler(d))
}

func respond(err error, w gohttp.ResponseWriter) {
	var res response
	if err != nil {
		res.Err = err.Error()
	}
	json.NewEncoder(w).Encode(&res)
}

func startLoggingHandler(d *driver.Driver) func(w gohttp.ResponseWriter, r *gohttp.Request) {
	return func(w gohttp.ResponseWriter, r *gohttp.Request) {
		var req StartLoggingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			gohttp.Error(w, err.Error(), gohttp.StatusBadRequest)
			return
		}
		if req.Info.ContainerID == "" {
			respond(errors.New("must provide container id in log context"), w)
			return
		}

		err := d.StartLogging(req.File, req.Info)
		respond(err, w)
	}
}

func stopLoggingHandler(d *driver.Driver) func(w gohttp.ResponseWriter, r *gohttp.Request) {
	return func(w gohttp.ResponseWriter, r *gohttp.Request) {
		var req StopLoggingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			gohttp.Error(w, err.Error(), gohttp.StatusBadRequest)
			return
		}
		err := d.StopLogging(req.File)
		respond(err, w)
	}
}

func capabilitiesHandler(d *driver.Driver) func(w gohttp.ResponseWriter, r *gohttp.Request) {
	return func(w gohttp.ResponseWriter, r *gohttp.Request) {
		json.NewEncoder(w).Encode(&CapabilitiesResponse{
			Cap: d.GetCapability(),
		})
	}
}

func readLogsHandler(d *driver.Driver) func(w gohttp.ResponseWriter, r *gohttp.Request) {
	return func(w gohttp.ResponseWriter, r *gohttp.Request) {
		var req ReadLogsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			gohttp.Error(w, err.Error(), gohttp.StatusBadRequest)
			return
		}

		stream, err := d.ReadLogs(req.Info, req.Config)
		if err != nil {
			gohttp.Error(w, err.Error(), gohttp.StatusInternalServerError)
			return
		}
		defer stream.Close()

		w.Header().Set("Content-Type", "application/x-json-stream")
		wf := ioutils.NewWriteFlusher(w)
		io.Copy(wf, stream)
	}
}
