# Build, flash and simulate the robotic arm firmware with arduino-cli.
#
#   make build     compile for the real board (ESP8266, needs config.h)
#   make upload    flash the real board (PORT=/dev/ttyUSB0)
#   make monitor   open the serial monitor
#   make sim       compile for ESP32 and run it in Wokwi (needs WOKWI_CLI_TOKEN)

ARDUINO_CLI ?= arduino-cli
WOKWI_CLI   ?= wokwi-cli
FQBN        ?= esp8266:esp8266:nodemcuv2
SIM_FQBN    ?= esp32:esp32:esp32
PORT        ?= /dev/tty.usbserial-0001
SIM_TIMEOUT ?= 60000

BUILD_DIR := build

.PHONY: build build-sim upload monitor sim clean

build: config.h
	$(ARDUINO_CLI) compile --fqbn $(FQBN) --output-dir $(BUILD_DIR)/esp8266 .

build-sim:
	$(ARDUINO_CLI) compile --fqbn $(SIM_FQBN) --output-dir $(BUILD_DIR)/esp32 \
		--build-property compiler.cpp.extra_flags=-DWOKWI_SIM .

upload: build
	$(ARDUINO_CLI) upload --fqbn $(FQBN) --port $(PORT) --input-dir $(BUILD_DIR)/esp8266 .

monitor:
	$(ARDUINO_CLI) monitor --port $(PORT) --config baudrate=115200

sim: build-sim
	$(WOKWI_CLI) --timeout $(SIM_TIMEOUT) --interactive simulation

config.h:
	@echo "config.h not found: copy config.example.h to config.h and fill it in" >&2
	@exit 1

clean:
	rm -rf $(BUILD_DIR)
