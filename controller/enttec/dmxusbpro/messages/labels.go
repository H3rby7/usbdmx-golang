package messages

const (
	// This message requests the Widget firmware to run the Widget bootstrap to enable reprogramming of the Widget firmware.
	LABEL_REPROGRAM_FIRMWARE_REQUEST = 1
	// This message programs one Flash page of the Widget firmware. The Flash pages must be programmed in order from first to last Flash page, with the contents of the firmware binary file.
	LABEL_PROGRAM_FLASH_PAGE_REQUEST = 2
	// The Widget sends this message to the PC on completion of the Program Flash Page request.
	LABEL_PROGRAM_FLASH_PAGE_REPLY = 2
	// This message requests the Widget configuration.
	LABEL_GET_WIDGET_PARAMS_REQUEST = 3
	// The Widget sends this message to the PC in response to the Get Widget Parameters request.
	LABEL_GET_WIDGET_PARAMS_REPLY = 3
	// This message sets the Widget configuration. The Widget configuration is preserved when the Widget loses power.
	LABEL_SET_WIDGET_PARAMS_REQUEST = 4
	// The Widget sends this message to the PC unsolicited, whenever the Widget receives a DMX or RDM packet from the DMX port, and the Receive DMX on Change mode is 'Send always'.
	LABEL_RECEIVED_DMX_PACKET = 5
	/*
		This message requests the Widget to periodically send a DMX packet out of the Widget DMX port at the configured DMX output rate. This message causes the widget to leave the DMX port direction as output after each DMX packet is sent, so no DMX packets will be received as a result of this request.

		The periodic DMX packet output will stop and the Widget DMX port direction will change to input when the Widget receives any request message other than the Output Only Send DMX Packet request, or the Get Widget Parameters request.
	*/
	LABEL_OUTPUT_ONLY_SEND_DMX_PACKET_REQUEST = 6
	// This message requests the Widget to send an RDM packet out of the Widget DMX port, and then change the DMX port direction to input, so that RDM or DMX packets can be received
	LABEL_SEND_RDM_PACKET_REQUEST = 7
	/*
		This message requests the Widget send a DMX packet to the PC only when the DMX values change on the
		input port. By default the widget will always send, if you want to send on change it must be enabled by
		sending this message. This message also reinitializes the DMX receive processing, so that if change of state
		reception is selected, the initial received DMX data is cleared to all zeros.
	*/
	LABEL_RECEIVE_DMX_ON_CHANGE = 8
	// The Widget sends one or more instances of this message to the PC unsolicited, whenever the Widget receives a changed DMX packet from the DMX port, and the Receive DMX on Change mode is 'Send on data change only'.
	LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET = 9
	// This message requests the Widget serial number, which should be the same as that printed on the Widget case.
	LABEL_WIDGET_GET_SERIAL_NUMBER_REQUEST = 10
	// The Widget sends this message to the PC in response to the Get Widget Serial Number request.
	LABEL_WIDGET_GET_SERIAL_NUMBER_REPLY = 10
	// This message requests the Widget to send an RDM Discovery Request packet out of the Widget DMX port, and then receive an RDM Discovery Response (see Received DMX Packet).
	LABEL_SEND_RDM_DISCOVERY_REQUEST = 11
)

const (
	// Smallest possible label-index to identify the message type
	SMALLEST_LABEL_INDEX = 1
	// Biggest possible label-index to identify the message type
	BIGGEST_LABEL_INDEX = 11
)
