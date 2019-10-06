# Cec2Mqt
Cec2Mqtt enables you to read the status and control your CEC enabled devices using MQTT.
Currently its supported features are:
* Reading the power status (on/off) of devices
* Powering on and off devices
* Reading which device is active
* Home Assistant integration for auto discovery

# Requirements
Cec2Mqtt currently works on the following hardware:
* Generic x64 (Intel / AMD) computers, ARMv7 (32 bit), and ARMv8 (64) bit devices using the Pulse-Eight HDMI-CEC adapter
* Raspberry Pi with the build in CEC support
Testing has been done both on a generic x64 computer using the Pulse-Eight HDMI-CEC adapter and on a Raspberry Pi.

## Installation
The easiest way to run Cec2Mqtt is by using the Docker images. As Cec2Mqtt is still under development there are only development
images available, called ``dev``. Use ``robertme/cec2mqtt:dev`` on generic hardware, and ``robertme/cec2mqtt:rpi-dev`` for Raspberry Pis.
The latest includes support for the build in CEC support of the Raspberry Pi while the former does not and only supports the
Pulse-Eight HDMI-CEC adapter.

### Generic
Running on generic hardware can be done using:
```console
docker run -v /path/to/data/directory:/data/cec2mqtt --device=/dev/ttyACM0 robertme/cec2mqtt:dev
```
``/dev/ttyACM0`` has to be replaced with the path to the Pulse-Eight HDMI-CEC adpater

When using docker-compose the following ``docker-compose.yaml`` can be used as a starting point:
```yaml
version: '3'

services:
  cec2mqtt:
    container_name: cec2mqtt
    image: robertme/cec2mqtt:dev
    volumes:
      - ./data:/data/cec2mqtt
    devices:
      - /dev/ttyACM0
    restart: unless-stopped
```


### Raspberry Pi
Running on the Raspberry Pi requires mapping of two devices and is done like so:
```console
docker run -v /path/to/data/directory:/data/cec2mqtt --device=/dev/vcsm --device=vchiq robetme/cec2mqtt:rpi-dev
```

When using docker-compose the following ``docker-compose.yaml`` can be used as a starting point:
```yaml
version: '2'

services:
  cec2mqtt:
    container_name: cec2mqtt
    image: robertme/cec2mqtt:rpi-dev
    volumes:
      - ./data:/data/cec2mqtt
    devices:
      - /dev/vcsm
      - /dev/vchiq
    restart: unless-stopped
```

## Configuration
Configuring Cec2Mqtt is done by a YAML file which must be created before the first start. The file must be
placed in the data directory and be called ``config.yaml``. Required options to configure are the MQTT host and base topic.
A minimal configuration file looks like this:
```yaml
mqtt:
  host: 1.2.3.4:1883
  base_topic: cec2mqtt
```

To enable the Home Assistant integration the following configuration must be added:
```yaml
home_assistant:
  enable: true
```
Enabling this integration is the recommended way to use Cec2Mqtt in combination with Home Assistant as it removes the requirement to manually
configure the entities in Home Assistant.

Optionally an MQTT state topic with birth and will message can be configured (also required when using Home Assistant integration).
```yaml
mqtt:
  state_topic: cec2mqtt/state
  birth_message: online
  will_message: offline
```

A complete example of the configuration is the following:
```yaml
mqtt:
    host: 1.2.3.4:1883
    username: User
    password: P@ssw0rd
    state_topic: cec2mqtt/state
    birth_message: online
    will_message: offline
    base_topic: cec2mqtt
home_assistant:
    enable: true
    discovery_prefix: homeassistant
```

### Device configuration
Devices which have been found in the CEC network can be configured as well. For this you **must** first stop Cec2Mqtt. When Cec2Mqtt is stopped you
can open the devices.yaml file in the data directory. Here you can change the ``mqtt_topic`` which is used in MQTT.

Optionally ``ignore`` can be set to ``true`` to completely ignore a device after which no Cec2Mqtt doesn't support this device anymore and
you can't read the state nor control the device.

Note that under normal operations you must never change any of the other values like, ``id``, ``physical_address``, ``vendor_id`` and ``osd`` 
as these are used by Cec2Mqtt to remember and look up the device.
