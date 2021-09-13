package utils

import (
	"flag"
)

//ElevatorID is the ID of this Elevator, is set when running execute with id flag. Defaults to 0.
var ELEVATOR_ID int

//ElevatorPort is the port number of the elevator server
var ELEVATOR_PORT int

func init() {
	flag.IntVar(&ELEVATOR_ID, "id", 0, "ID of this Elevator")
	flag.IntVar(&ELEVATOR_PORT, "port", 15657, "Port of the Elevator")
	flag.Parse()
}



//Elevator Settings
const (

	//FLOOR_NUM is the number of floors
	FLOOR_NUM = 4

	// ORDER_TYPE_NUM is the number of order types
	ORDER_TYPE_NUM = 3

	// ELEVATOR_MAX_NUM is max number of elevators in system
	ELEVATOR_MAX_NUM = 3

	// DOOR_OPEN_TIME is how long door is open on floor arrival in seconds
	DOOR_OPEN_TIME = 3

	// TRAVEL_TIME is the time it takes in seconds for the elevator to move for one floor to another
	TRAVEL_TIME = 3

	// MAX_OBSTRUCT_TIME is time in seconds before elevator is considered unavailable
	MAX_OBSTRUCT_TIME = 7

	// MAX_TRAVEL_TIME is the longest time in seconds between floors
	MAX_TRAVEL_TIME = 6

	// MAX_DECIDE_TIME is the max time in milliseconds for agreeing on an order
	MAX_DECIDE_TIME = 500

	// ADD ELEVATOR SETTINGS HERE
)

//Connection Settings
const (

	//CONNECTION_DATA_PORT is the UDP port used to send data packets
	CONNECTION_DATA_PORT = 12067

	//CONNECTION_CHECK_PORT is the UDP port used to send awake messages
	CONNECTION_CHECK_PORT = 12068

	//CONNECTION_ACK_PORT is the UDP port used to send Ack messages
	CONNECTION_ACK_PORT = 12069

	//CONNECTION_CHECK_INTERVAL is the interval between awake messages in milliseconds
	CONNECTION_CHECK_INTERVAL = 100

	//CONNECTION_CHECK_TRESHOLD is the number of awake messages that has to be lost for connection to be considered lost. Dropout latency = Interval * Treshold
	CONNECTION_CHECK_TRESHOLD = 10

	//ACK_TIMEOUT is how long network module waits for ack. If all acks are not in before timeout, the packet is resent.
	ACK_TIMEOUT = 15

	//ACK_ATTEMPTS is the number of times the network module tries to resend a packet after timout.
	ACK_ATTEMPTS = 30

	//RX_PACKET_REGISTER_LENGTH is the number of packet IDs stored to prevent duplicates.
	RX_PACKET_REGISTER_LENGTH = 36
	
)
