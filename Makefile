PROJECT_NAME ?= iot-motion-sensor-go

DOCKER_SERVICE_GO ?= go
DOCKER_COMPOSE_RUN = docker-compose run --rm

STAGE ?= dev
TEST_TARGET ?= ./...

RASPI_IP ?= 192.168.3.16

all:
	$(MAKE) install

.env: .env.sample
	cp .env.sample .env
	@echo "An .env file has been created. Please correct."

docker-build: .env
	docker-compose build

install: docker-build
	$(MAKE) build

down:
	docker-compose down

.PHONY: build
build lint clean exec:
	$(DOCKER_COMPOSE_RUN) $(DOCKER_SERVICE_GO) make -f Makefile_go $@

test:
	$(DOCKER_COMPOSE_RUN) $(DOCKER_SERVICE_GO) make -f Makefile_go $@ TEST_TARGET=$(TEST_TARGET)

.PHONY: deploy
deploy: build/deploy/cmd/${PROJECT_NAME}
	scp $< pi@$(RASPI_IP):~/
