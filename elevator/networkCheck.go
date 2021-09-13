package elevator

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"time"

	"./conn"
	"./utils"
)

type connCheckPacket struct {
	ElevatorID int
}

// Connection check function starts both receiving, sending and handling the connection checking.
func ConnectionCheck(connect chan<- ConnectionEvent) {
	connectionStatus := make([]bool, utils.ELEVATOR_MAX_NUM)
	var timer [utils.ELEVATOR_MAX_NUM]*time.Timer
	recieve := make(chan int)

	receivedFlag := make([]bool, utils.ELEVATOR_MAX_NUM)
	consecutiveLosses := make([]int, utils.ELEVATOR_MAX_NUM)

	// Create a select case for receiving an awake message
	selectCases := make([]reflect.SelectCase, utils.ELEVATOR_MAX_NUM+1)
	selectCases[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(recieve),
	}

	// Create a select case for timeout channel of timers for each elevator
	for i := 0; i < utils.ELEVATOR_MAX_NUM; i++ {
		timer[i] = time.NewTimer(utils.CONNECTION_CHECK_INTERVAL * time.Millisecond)
		timer[i].Stop()
		selectCases[1+i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(timer[i].C),
		}
	}

	//Start connection Check sending
	go connectionCheckSend()

	//Start Receiving Connection checks
	go connectionCheckRecieve(recieve)

	for {
		chosen, value, _ := reflect.Select(selectCases)
		switch chosen {
		case 0:
			// Awake message received for an elevator
			ElevatorID := int(value.Int())

			if ElevatorID != utils.ELEVATOR_ID {
				// Set received flag to true, and clear consecutive losses
				receivedFlag[ElevatorID] = true
				consecutiveLosses[ElevatorID] = 0
				if !connectionStatus[ElevatorID] {
					// If disconnected set to connected
					timer[ElevatorID].Stop()
					timer[ElevatorID].Reset(utils.CONNECTION_CHECK_INTERVAL * time.Millisecond)
					d := ConnectionEvent{ElevatorID, true}
					connectionStatus[ElevatorID] = true
					connect <- d

				}
			}
		default:
			ElevatorID := chosen - 1
			timer[ElevatorID].Reset(utils.CONNECTION_CHECK_INTERVAL * time.Millisecond)
			if !receivedFlag[ElevatorID] {
				// If the recived flag is not set, add a to a consecutive loss.
				// If consecutive losses reaches teshold set to disconnect.
				consecutiveLosses[ElevatorID]++

				if (consecutiveLosses[ElevatorID]) == utils.CONNECTION_CHECK_TRESHOLD {
					d := ConnectionEvent{ElevatorID, false}
					connectionStatus[ElevatorID] = false
					connect <- d
					timer[ElevatorID].Stop()
				}
			}
			// Set received flag to false
			receivedFlag[ElevatorID] = false

		}
	}
}

func connectionCheckSend() {

	d := connCheckPacket{ElevatorID: utils.ELEVATOR_ID}
	jsonstr, err := json.Marshal(d)
	utils.CheckError(err)

	conn := conn.DialBroadcastUDP(utils.CONNECTION_CHECK_PORT)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", utils.CONNECTION_CHECK_PORT))
	for {

		conn.WriteTo(jsonstr, addr)
		time.Sleep(utils.CONNECTION_CHECK_INTERVAL * time.Millisecond)
	}
}

func connectionCheckRecieve(r chan<- int) {
	var buf [16]byte
	conn := conn.DialBroadcastUDP(utils.CONNECTION_CHECK_PORT)
	for {
		n, _, e := conn.ReadFrom(buf[0:])
		utils.CheckError(e)
		var packet connCheckPacket
		json.Unmarshal(buf[0:n], &packet)
		// fmt.Println("Receive", packet.ElevatorID)
		r <- packet.ElevatorID
	}
}
