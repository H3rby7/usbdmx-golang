# Enttec USB DMX Pro

Some Notes from the Enttec DMX USB Pro - API Document @v1.44 

- [Enttec USB DMX Pro](#enttec-usb-dmx-pro)
  - [IN and OUT](#in-and-out)
  - [Message-Format](#message-format)
    - [Labels](#labels)
- [Glossary](#glossary)

## IN and OUT

DMX USB Pro has been designed to either receive or send a DMX stream at any one time, not both.

The DMX USB Pro’s input and output ports are physically connected to each other, therefore trying to recieve and send DMX streams would cause data degradation and flickering. Having the DMX IN and DMX OUT options it means that it can be integrated as part of a DMX chain when set to receive DMX data (in the same way you can daisy chain DMX in and out of a lighting fixture).

[Source](https://support.enttec.com/support/solutions/articles/101000395672-usb-dmx-input-output)

## Message-Format

Size in Bytes | Description
--- | ---
1 | Start of message delimiter, 0x7E.
1 |Label to identify type of message. See [Labels](#labels)
1 | Data length LSB. Valid range for data length is 0 to 600.
1 | Data length MSB.
*[data_length]* | Payload bytes (byte at index `1` contains channel `1`. So byte `0` is unused)
1 | End of message delimiter, 0x7E.

### Labels

Label # | Title (in API description)
--- | ---
1 | Reprogram Firmware Request
2 | Program Flash Page (Request/Reply)
3 |  Get Widget Parameters (Request/Reply)
4 | Set Widget Parameters Request
5 | Received DMX Packet
6 |  Output Only Send DMX Packet Request
7 | Send RDM Packet Request
8 | Receive DMX on Change
9 | Received DMX Change Of State Packet
10 | Get Widget Serial Number (Request/Reply)
11 | Send RDM Discovery Request

# Glossary

Term | Explanation
--- | ---
LSB | Least Significant Bit
MSB | Most Significant Bit
