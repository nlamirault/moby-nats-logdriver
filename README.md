# Nats Log Driver

[![Build Status](https://travis-ci.org/nlamirault//moby-nats-logdriver.svg?branch=master)](https://travis-ci.org/nlamirault/moby-nats-logdriver)

Moby/Docker support log driver plugins which allows people to add extra functionality to handle logs coming out of docker containers.
This plugin allows users to route all Moby/Docker logs to Nats.

***This plugin requires at least Docker version 17.05***


## Installation

### From Dockerhub:

Install the plugin but add the --disable flag so it does not start immediately. The nats brokers must be set first.
```
docker plugin install --disable mickyg/nats-logdriver:latest
```
Set the Nats endpoint and configure the plugin as per the configuration section.
In the example below the host 192.168.0.1 is a Nats endpoint
```
docker plugin set nlamirault/nats-logdriver:latest NATS_ADDR="192.168.0.1:9092"
```
Then enable the plugin
```
docker plugin enable nlamirault/nats-logdriver:latest
```
Now it's installed! To use the log driver for a given container, use the `--logdriver` flag. For example, to start the hello-world
container with all of it's logs being sent to Nats, run the command:
```
docker run --log-driver nlamirault/nats-logdriver:latest hello-world
```


### From source

Clone the project
```
git clone https://github.com/MickayG/moby-nats-logdriver.git
cd moby-nats-logdriver
```
Build the plugin and install it
```
make install
```
Set the NATS_ADDR variable. In the example below the host 192.168.0.1 is a Nats server.
```
docker plugin set nlamirault/nats-logdriver:latest NATS_ADDR="192.168.0.1:9092"
```
Enable the plugin:
```
make enable
```
Now test it! Connect to your broker and consume from the "dockerlogs" topic
(Topic can be changed via environment variable, see below). Then launch a container:
```
docker run --log-driver mickyg/nats-logdriver:latest hello-world
```


## Configuration

Docker logdriver plugins can be configured on global/plugin basis. The below configurations when applied act on all containers where the logdriver plugin is used.

Once the plugin has been installed, you can modify the below arguments with the command
```
docker plugin set nats-logdriver <OPTION>=<VALUE>
```
For example, to change the subject to "logs"
```
docker plugin set nats-logdriver NATS_SUBJECT=logs
```

| Option | Description | Default |
| -------|-------------| --------|
| NATS_ADDR	| **(Required)** Nats servers address (like "nats://localhost:4322") | |
| NATS_SUBJECT | **(Required)** Subject for Nats messaging | |
| LOG_LEVEL	| Log level of the internal logger. Options: debug, info, warn, error | info |


## Output format

Each log message will be written to a single Nats message.

| Field | Description |
| Line |The log message itself |
| Source | Source of the log message as reported by docker |
| Timestamp | Timestamp that the log was collected by the log driver |
| ContainerName | Name of the container that generated the log message |
| ContainerId | Id of the container that generated the log message |
| ContainerImageName | Name of the container's image |
| ContainerImageId | ID of the container's image |
| Err |	Usually null, otherwise will be a string containing and error from the logdriver |
