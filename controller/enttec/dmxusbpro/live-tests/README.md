# Live Tests

Conducted with two widgets both connected to the computer via USB and to each other using DMX.

- [Live Tests](#live-tests)
  - [RW Fast](#rw-fast)
  - [Simulate Fader Up](#simulate-fader-up)


## RW Fast

Writer sends changing RGB commands.

    go run controller\enttec\dmxusbpro\live-tests\rw-fast\main.go --writer=COM6 --reader=COM5 --read-interval=5 --write-interval=5

## Simulate Fader Up

Simulates a fader (controlling multiple DMX channels) being moved up.

    go run controller\enttec\dmxusbpro\live-tests\simulate-fader-up\main.go --writer=COM6 --reader=COM5 --read-interval=5 --write-interval=5