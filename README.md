# securityblanket -- DIY home security project

This project aims to receive, store, process, analyze & alert based on
updates from home security sensors, remind about battery changes, etc.

Current supported hardware is Honeywell 5800 series (compatible) RF
sensors. This includes door/window sensors, motion detection,
fire/smoke/heat, and more. As far as I'm aware, this is the only open
source project that correctly decodes the events for multiple sensor
types.

Honeywell 5800 RF transmissions are received with a SDR usb stick and
by running `rtl_433` as a subprocess, at least for now. Other sensor
standards (Z-Wave, Zigbee etc) would be nice to support, and we'd like
to have a modular interface where you can track any kind of a sensor
-- but that needs people with other kinds of sensors to pitch in.

Intended software integrations, at the minimum: Home Assistant,
Prometheus.

We may dabble with video cameras & their motion detection, too.

Probably something to drive a siren (talk over wifi to
arduino/rpizw that controls a relay?).

State is stored in SQLite, and hardware requirements are intended to
be modest enough to run on a Raspberry Pi.


## Current status

Receives Honeywell 5800 series transmissions, understands the content,
records them in SQLite.

To get involved at this stage, you are expected to understand
programming and SQLite. We're not quite ready for end users.

Generated code is not currently committed to the git repo. That'll
happen after things stabilize.


## Building

You'll need the Go compiler suite installed. Minimum version is listed
in the `go.mod` file.

```
git clone https://github.com/tv42/securityblanket
cd securityblanket
go generate ./...
go build ./cmd/...
```


## Demo of what exists today

Last few sensor trips, and some of the information known about the
sensors:

```
$ sqlite3 securityblanket.sqlite
sqlite> select * from honeywell5800_trips order by id desc limit 2;
id          sensor      loop        trippedBy   clearedBy
----------  ----------  ----------  ----------  ----------
256         987654      1           1169        1170
255         987654      1           1162        1163

sqlite> select * from honeywell5800_sensors where id=987654;
id          created                  model        description
----------  -----------------------  -----------  ------------------
987654      2020-02-11T05:30:25.334  5800PIR-RES  living room motion

sqlite> select * from honeywell5800_model_loops where model='5800PIR-RES';
model        loop        kind             factoryLabel  factoryNormallyOpen  typicallyUnused
-----------  ----------  ---------------  ------------  -------------------  ---------------
5800PIR-RES  1           motion detector                0                    0
5800PIR-RES  4           tamper                         0                    0

sqlite> select * from honeywell5800_updates where id in (1169, 1170);
id          time                                 channel     sensor      event
----------  -----------------------------------  ----------  ----------  ----------
1169        2020-02-12T12:32:23.419910853-07:00  8           987654      128
1170        2020-02-12T12:32:26.565627184-07:00  8           987654      0

sqlite>
```

The Honeywell 5800 series protocol does not carry information about
what kind of a sensor is transmitting. The system adds rows to the
`honeywell5800_sensors` table, but you need to set the `model` column
in order for the system to understand what the events mean. See table
`honeywell5800_models` for currently recognized models.


## Roadmap

- Web UI, general editability of your sensors.
- Home Assistant integration (still in learning phase).
- Alerting. Most likely native web push alerts, and a good story about
  modularity to integrate everything else.
- Acknowledging alerts.
- Armed states, as in don't alert when you're at home.
- Notify on low battery (information is already in database).
- Notify on missed heartbeat (last heartbeat is already in database).
- Documentation.
- Easier learning curve for people without a SQL background. Goal: If
  you're comfortable with DIY and RPi, it'll be a breeze.
- Maybe reimplement Manchester decoding and use librtl directly.
- Make it (even) more robust, dead letter queues instead of bailing
  out on errors etc.
