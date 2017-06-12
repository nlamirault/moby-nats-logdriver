# Copyright (C) 2017 Nicolas Lamirault <nicolas.lamirault@gmail.com>

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.8-alpine
MAINTAINER Nicolas Lamirault <nicolas.lamirault@gmail.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

# RUN apk add --no-cache \
#     ca-certificates

COPY . /go/src/github.com/nlamirault/moby-nats-logdriver

RUN set -x \
    && cd /go/src/github.com/nlamirault/moby-nats-logdriver \
    && go build -o /usr/bin/moby-nats-logdriver . \
    && echo "Build complete."

ENTRYPOINT ["/usr/bin/moby-nats-logdriver"]