# golang broadlink rm

## Overview

This is an app (`rmproxy`) and a library which lets you interact with the Broadlink RM Pro+ infrared blaster.

It is based on [broadlink-rm-http](https://github.com/TheAslera/broadlink-rm-http) and the [broadlinkjs-rm](https://github.com/lprhodes/broadlinkjs-rm) library.

## Docker Support

The app can run from within a docker container. To do that, execute `make image` to build the docker image, followed by `make runcontainer` to run the container.

Before you run the container, you will probably want to customize the JSON files located in the `json` directory.

The JSON files are loosely based on the broadlink-rm-http's Javascript config files. More information on the configuration can be found on [broadlink-rm-http's github page](https://github.com/TheAslera/broadlink-rm-http).

If you're running docker on linux, discovery of Broadlink devices will only work if you're running the container using the host network mode (`--network host` on the `docker run` command line). This workaround doesn't work on Docker for Mac.

You can still run `rmproxy` without discovery working if you enter the device configuration (IP address, key, id, and an optional MAC address) in a JSON file. Refer to the section below for more details on how to do this. You will need to run on a setup with discovery working (either by running the container using the host network mode or by running `rmproxy` outside of a container) in order to get the key and ID.

## Device Config File

`rmproxy` will send a UDP broadcast packet when it starts up to discover all the Broadlink devices on the network.

If it finds a supported Broadlink device, it will send an authentication packet to the device. The device should reply with an encryption key and an ID. `rmproxy` will print out the new encryption key and the ID on `stdout`.

If you wish to skip this discovery process, you can specify the device's attributes in a JSON file and launch `rmproxy` with the `-deviceconfig` command line option or the `DEVICECONFIG` environment variable. You can also include a `-skipdiscovery` command line option or a `SKIPDISCOVERY` environment variable.

A sample device config JSON file can be found at `json/devices_sample.json`.

## Web Remote Control

A simple web remote interface is available at `http://localhost:8080/remote/KEY/`.

## Credits

The remote control icons were downloaded from <https://icons8.com/> and <https://material.io/tools/icons/>.

A large part of the code was ported over from <https://github.com/TheAslera/broadlink-rm-http> and <https://github.com/lprhodes/broadlinkjs-rm>.

The [blog post by Ipsum Domus](https://blog.ipsumdomus.com/broadlink-smart-home-devices-complete-protocol-hack-bc0b4b397af1) really helped in showing how the protocol works.

Inspiration for the remote control web layout was taken from <https://www.youtube.com/watch?v=X1SNkEZW5h8>.