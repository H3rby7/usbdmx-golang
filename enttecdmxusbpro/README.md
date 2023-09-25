# Enttec USB DMX Pro

- [Enttec USB DMX Pro](#enttec-usb-dmx-pro)
  - [Message-Format](#message-format)
    - [Labels](#labels)
- [Glossary](#glossary)

## Message-Format

Size in Bytes | Description
--- | ---
1 | Start of message delimiter, 0x7E.
1 |Label to identify type of message. See [Labels](#labels)
1 | Data length LSB. Valid range for data length is 0 to 600.
1 | Data length MSB.
*[data_length]* | Payload bytes
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
