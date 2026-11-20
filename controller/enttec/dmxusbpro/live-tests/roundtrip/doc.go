/*
Package roundtrip holds tests that need real hardware:
two Enttec DMX USB Pro widgets, both connected via USB, and to each other via DMX.

One widget writes DMX, the other one reads it, and the tests confirm the reader receives what the writer sent.

The tests only build with the 'hardware' build tag, so they don't run in CI or with a plain 'go test ./...'.
Run them from the repository root with:

	go test -tags hardware -v -count=1 ./controller/enttec/dmxusbpro/live-tests/roundtrip/ -args -writer=COM8 -reader=COM7

Mind that the writer drives the DMX line, so any fixture connected to it will react.
*/
package roundtrip
