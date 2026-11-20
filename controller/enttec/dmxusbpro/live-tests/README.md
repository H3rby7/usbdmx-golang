# Live Tests

Conducted with two widgets both connected to the computer via USB and to each other using DMX.

- [Live Tests](#live-tests)
  - [Round Trip](#round-trip)
  - [RW Fast](#rw-fast)
  - [Simulate Fader Up](#simulate-fader-up)
    - [Observations](#observations)

## Round Trip

An automated test (`go test`), unlike the other live tests, which only log.
The writer sends several DMX states (all channels, a single channel, all zero) and the test confirms each one with the reader,
once with the reader in 'changes only' mode and once in 'send always' mode.
It also checks that `Disconnect` stops a reader that is waiting for data on a static DMX input.

The test needs the `hardware` build tag, so it doesn't run in CI or with a plain `go test ./...`.
Run it from the repository root, `-writer` and `-reader` default to `COM8` and `COM7`:

    go test -tags hardware -v -count=1 ./controller/enttec/dmxusbpro/live-tests/roundtrip/ -args -writer=COM8 -reader=COM7

Further flags (pass them after `-args`):

Flag | Default | Meaning
--- | --- | ---
`-channels` | `512` | number of DMX channels to write and confirm (1 to 512)
`-read-interval` | `5` | read interval of the reader in milliseconds
`-timeout` | `3s` | how long the reader may take to confirm a written state
`-stop-timeout` | `5s` | how long the reader may take to stop after `Disconnect`
`-baud` | `57600` | baud rate for both widgets

*Note: any fixture connected to the DMX line will react, the output is cleared at the end of each test.*

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
                    1 |                     1 |            no |                  30-70 |          2/1
                    1 |                     1 |           yes |                  10-30 |          2/1
                    5 |                     5 |            no |                    ~50 |          2/1
                    5 |                     5 |           yes |                  20-30 |          3/2
                   12 |                    12 |            no |                  30-70 |          5/3
                   12 |                    12 |           yes |                  20-30 |          3/2
                   16 |                    16 |            no |                  40-80 |          1/1
                   16 |                    16 |           yes |                    ~40 |          1/1
                   25 |                    25 |            no |                    ~70 |          5/6
                   25 |                    25 |           yes |                  40-70 |          1/1
