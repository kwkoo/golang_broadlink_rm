# golang broadlink rm

## Overview

This repository consists of several components:

1. `broadlinkrm` (`src/github.com/kwkoo/broadlinkrm`) - A Go library designed to communicate with the Broadlink RM Pro+ infrared blaster and the Broadlink SP2 wifi-enabled power outlet. It is based on [broadlinkjs-rm](https://github.com/lprhodes/broadlinkjs-rm).

2. `demo` (`src/github.com/kwkoo/broadlinkrm/cmd/demo`) - A simple web app which demonstrates how to use `broadlinkrm`. Access <http://localhost:8080/learn> to put the RM Pro into learning mode. After it learns the remote code, access <http://localhost:8080/> to emit the learned code.

3. `rmproxy` (`src/github.com/kwkoo/broadlinkrm/cmd/rmproxy`) - A web app based on [broadlink-rm-http](https://github.com/TheAslera/broadlink-rm-http). It lets you put the RM Pro into learning mode by accessing <http://localhost:8080/learn/KEY/IP_ADDRESS>. You can also make `rmproxy` emit a remote code by accessing <http://localhost:8080/execute/KEY/ROOM/COMMAND>. Lastly, you `rmproxy` has a web-based remote control that is accessed at <http://localhost:8080/remote/KEY/>. The remote module will require customization to suit your deployment. `rmproxy` also supports macros, which let you emit a sequence of remote commands with a single URL. You can send a macro by accessing <http://localhost:8080/macro/KEY/MACRO_NAME>.

4. `macrobuilder` (`src/github.com/kwkoo/broadlinkrm/cmd/macrobuilder`) - A web app that takes in the same configuration files as `rmproxy` and lets you build a macro and generates JSON that you can copy and paste into a file that `rmproxy` can parse.


## `rmproxy` Docker Support

`rmproxy` can run from within a docker container. To do that, execute `make image` to build the docker image, followed by `make runcontainer` to run the container.

Before you run the container, you will probably want to customize the JSON files located in the `json` directory.

The JSON files are loosely based on the broadlink-rm-http's Javascript config files. More information on the configuration can be found on [broadlink-rm-http's github page](https://github.com/TheAslera/broadlink-rm-http).

If you're running docker on linux, discovery of Broadlink devices will only work if you're running the container using the host network mode (`--network host` on the `docker run` command line). This workaround doesn't work on Docker for Mac.

You can still run `rmproxy` without discovery working if you enter the device configuration (IP address, key, id, and an optional MAC address) in a JSON file. Refer to the section below for more details on how to do this. You will need to run on a setup with discovery working (either by running the container using the host network mode or by running `rmproxy` outside of a container) in order to get the key and ID.


## `rmproxy` Device Config File

`rmproxy` will send a UDP broadcast packet when it starts up to discover all the Broadlink devices on the network.

If it finds a supported Broadlink device, it will send an authentication packet to the device. The device should reply with an encryption key and an ID. `rmproxy` will print out the new encryption key and the ID on `stdout`.

If you wish to skip this discovery process, you can specify the device's attributes in a JSON file and launch `rmproxy` with the `-deviceconfig` command line option or the `DEVICECONFIG` environment variable. You can also include a `-skipdiscovery` command line option or a `SKIPDISCOVERY` environment variable.

A sample device config JSON file can be found at `json/devices_sample.json`.


## `rmproxy` Web Remote Control

A simple web remote interface is available at `http://localhost:8080/remote/KEY/`.

Please note that this probably won't work for you out of the box.

## Macros

`rmproxy` supports macros. Macros let you send a sequence of remote codes with a single URL.

An example of the format expected can be found at `json/macros_sample.json`.

There are 2 types of macro instructions.

1. `sendcommand` - Tells `rmproxy` to emit a remote code. It's in the following format: `sendcommand ROOM COMMAND`

2. `pause` - Tells `rmproxy` to pause before sending the next command. It's in the following format: `pause INTERVAL` where `INTERVAL` is an integer specifying the number of milliseconds to pause.

If you wish to create a large number of macros, it may make sense to use `macrobuilder` to generate the JSON for those macros. `macrobuilder` uses the same rooms JSON file and commands JSON file as `rmproxy`.

## Quickstart

The easiest way to get started is to run it within Docker.

Before you get started, head into the `json` directory and make copies of each JSON file. The Dockerfile expects to find `commands.json`, `devices.json`, `macros.json`, and `rooms.json` in the json directory. You can change these default file names in `env.list`.

If you already have Docker installed, execute `make image` to create the Docker image, followed by `make runcontainer` to run the Docker container.

At this point, there should be something listening on port 8080. It’ll send out a broadcast packet to see if there are any Broadlink RM devices on the network. If it gets a response, it’ll be logged to the console.

Use a web browser to access <http://localhost:8080/learn/123/IPADDRESS> where `IPADDRESS` ss the IP address of the Broadlink RM Pro. This should put the Broadlink RM Pro into learning mode. Point an infrared remote at the Broadlink RM Pro and press a button on the remote. The remote code should be printed on the web browser.

You can then copy the learned commands to `commands.json`. You’ll need to rebuild the Docker image after you change the JSON files. Then run a container based on the new image.

Once the new container is running, access <http://localhost:8080/execute/123/ROOMNME/COMMANDNAME> to get the Broadlink RM Pro to emit the command.

You can get more info on how to setup the JSON files on [broadlink-rm-http’s page](https://github.com/TheAslera/broadlink-rm-http).


## Credits

The remote control icons were downloaded from <https://icons8.com/> and <https://material.io/tools/icons/>.

A large part of the code was ported over from <https://github.com/TheAslera/broadlink-rm-http> and <https://github.com/lprhodes/broadlinkjs-rm>.

The [blog post by Ipsum Domus](https://blog.ipsumdomus.com/broadlink-smart-home-devices-complete-protocol-hack-bc0b4b397af1) really helped in showing how the protocol works.

Inspiration for the remote control web layout was taken from <https://www.youtube.com/watch?v=X1SNkEZW5h8>.
