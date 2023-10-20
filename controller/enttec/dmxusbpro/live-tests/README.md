# Live Tests

Conducted with two widgets both connected to the computer via USB and to each other using DMX.

- [Live Tests](#live-tests)
  - [RW Fast](#rw-fast)
  - [Simulate Fader Up](#simulate-fader-up)
    - [Observations](#observations)


## RW Fast

Writer sends changing RGB commands.

    go run controller\enttec\dmxusbpro\live-tests\rw-fast\main.go --writer=COM6 --reader=COM5 --read-interval=5 --write-interval=5

## Simulate Fader Up

Simulates a fader (controlling multiple DMX channels) being moved up.

    go run controller\enttec\dmxusbpro\live-tests\simulate-fader-up\main.go --writer=COM6 --reader=COM5 --read-interval=5 --write-interval=5 --changes-only=true

### Observations

Running the 'fader-up' live test with different parameters.

Legend:

Title | Meaning
--- | ---
Interval (R) [millis] | read interval in milliseconds
Interval (W) [millis] | write interval in milliseconds
Changes only? | are we using `DMX Change Of State Packet`s?
rountrip-time [millis] | time (in milliseconds) taken from `about-to-write` to `read-back-in`.
Writes/Reads | Ratio between writes and reads. Numbers > 1 mean we have more writes than reads and vice versa

*`roundtrip-time` was calculated for random entries (e.g. channel 1 being set to 120), using the log timestamp difference between WRITER and READER (e.g. the reader received 120)*

Experiments:

Interval (R) [millis] | Interval (W) [millis] | Changes only? | rountrip-time [millis] | Writes/Reads
--------------------- | --------------------- | ------------- | ---------------------- | ------------
                    1 |                     1 |            no |                  50-60 |            3
                    1 |                     1 |           yes |                  50-60 |            3
                    5 |                     5 |            no |                  50-80 |            3
                    5 |                     5 |           yes |                     30 |            3
                   12 |                    12 |            no |                     30 |            3
                   12 |                    12 |           yes |                     30 |            3
                   25 |                    25 |            no |                     70 |            1
                   25 |                    25 |           yes |                     40 |            1
