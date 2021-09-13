package elevator

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"time"

	"./conn"
	"./log"
	"./utils"
)

type ackPacket struct {
	ElevatorID int
	PacketID   int
}

type AckRoutine struct {
	PacketID      int
	AckReceivedCh chan int
}

type filterFunction func(v reflect.Value) bool

// Transmitter Starts transmitting data as well as starting routines used to handle ack receiving.
// data received through provided chans will be sent if they the provided filter functiion returns
// true evaluating said data. Transmitter will expect to receive acks from connected elevators.
// It keeps track of connected elevators through the provided connectionEvent Channel.
func Transmitter(filter filterFunction, connectionFail chan ConnectionEvent, chans ...interface{}) {
	checkArgs(chans...)
	id := 0
	n := 0

	for range chans {
		n++
	}
	TXCh := make(chan []byte)
	addAckRoutineCh := make(chan AckRoutine)
	doneAckCh := make(chan int)
	selectCases := make([]reflect.SelectCase, n+1)
	typeNames := make([]string, n)
	availableElevators := make(map[int]interface{})

	// Create select case for connection fail and for each data channel
	selectCases[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(connectionFail),
	}

	for i, ch := range chans {
		selectCases[i+1] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		}
		typeNames[i] = reflect.TypeOf(ch).Elem().String()
	}

	go recieveAck(addAckRoutineCh, doneAckCh)
	go TX(TXCh)
	for {
		chosen, value, _ := reflect.Select(selectCases)
		switch chosen {
		case 0:
			// keep track of connected elevators on connectionEvent
			ElevatorID := int(value.Field(0).Int())
			Connect := value.Field(1).Bool()

			if ElevatorID != utils.ELEVATOR_ID {
				if Connect {
					availableElevators[ElevatorID] = nil
				} else {
					delete(availableElevators, ElevatorID)
				}
			}

		default:
			if filter(value) {
				// If data passes the filter test, create packet id, send data and handle ack routine
				jsonstr, _ := json.Marshal(value.Interface())
				payload := typeTaggedJSON{
					TypeId: typeNames[chosen-1],
					JSON:   jsonstr,
				}
				p, _ := json.Marshal(payload)
				packetID := (utils.ELEVATOR_ID << 8) + (id & 255)
				id++
				packet := dataPacket{packetID, p}
				ttj, err := json.Marshal(packet)
				utils.CheckError(err)
				log.PrintDbg("expecting ack from", len(availableElevators), "Elevators")
				go handleSend(packetID, ttj, len(availableElevators), TXCh, addAckRoutineCh, doneAckCh)
			}
		}
	}
}

// TX writes data sent through channel to the udp port
func TX(ch <-chan []byte) {
	conn := conn.DialBroadcastUDP(utils.CONNECTION_DATA_PORT)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", utils.CONNECTION_DATA_PORT))
	for {
		packet := <-ch
		conn.WriteTo(packet, addr)
		time.Sleep(10 * time.Millisecond)
	}
}

// RXack receives ack packets on ack port and send it through the AckCh channel
func RXack(AckCh chan<- ackPacket) {
	var buf [64]byte

	conn := conn.DialBroadcastUDP(utils.CONNECTION_ACK_PORT)
	for {
		var packet ackPacket
		n, _, e := conn.ReadFrom(buf[0:])
		utils.CheckError(e)
		json.Unmarshal(buf[0:n], &packet)
		if (packet.PacketID>>8) == utils.ELEVATOR_ID && packet.ElevatorID != utils.ELEVATOR_ID {
			AckCh <- packet
		}
	}
}

// ReceiveAck receives acks from ackRX and passes them on to the correct handleSend() routine.
// Channels to handleSend() routine is added through addAckCh channel.
func recieveAck(addAckCh <-chan AckRoutine, doneAckCh <-chan int) {
	pending := make(map[int]AckRoutine)
	ackRX := make(chan ackPacket)
	go RXack(ackRX)
	for {
		select {
		case p := <-ackRX:
			_, exist := pending[p.PacketID]
			if exist {
				go func(ch chan<- int, elevatorID int) {
					ch <- elevatorID
				}(pending[p.PacketID].AckReceivedCh, p.ElevatorID)
			} else {
				log.PrintDbg("RecieveAck that didnt exist p: ", p.PacketID)
			}

		case add := <-addAckCh:

			pending[add.PacketID] = add
		case PacketID := <-doneAckCh:
			delete(pending, PacketID)
		}
	}
}

// handleSend handles the transmision of a packet. If the expected ack messages is not
// received within timout, packet i resent, until max attempt is reached
func handleSend(packetID int, packet []byte, numElevators int, sendCh chan<- []byte, addAckCh chan<- AckRoutine, doneAckCh chan<- int) {
	ackReg := make(map[int]interface{})
	ackCh := make(chan int)
	attempts := 0
	reciever := AckRoutine{packetID, ackCh}
	addAckCh <- reciever
	sendCh <- packet
	timeout := time.NewTimer(utils.ACK_TIMEOUT * time.Millisecond)

	if numElevators == 0 {
		return
	}

	for {
		select {
		case <-timeout.C:
			if attempts < utils.ACK_ATTEMPTS {
				attempts++
				timeout.Reset(utils.ACK_TIMEOUT * time.Millisecond)
				sendCh <- packet
				log.PrintDbg("Packet not acknowledged, resending", packetID)
			} else {
				log.PrintErr("Packet not acknowledged", packetID)
			}
		case ElevatorID := <-ackCh:
			ackReg[ElevatorID] = nil
			if len(ackReg) >= numElevators {
				doneAckCh <- packetID
				return
			}
		}
	}
}

// Checks that args to Tx'er/Rx'er are valid:
//  All args must be channels
//  Element types of channels must be encodable with JSON
//  No element types are repeated
// Implementation note:
//  - Why there is no `isMarshalable()` function in encoding/json is a mystery,
//    so the tests on element type are hand-copied from `encoding/json/encode.go`
func checkArgs(chans ...interface{}) {
	n := 0
	for range chans {
		n++
	}
	elemTypes := make([]reflect.Type, n)

	for i, ch := range chans {
		// Must be a channel
		if reflect.ValueOf(ch).Kind() != reflect.Chan {
			panic(fmt.Sprintf(
				"Argument must be a channel, got '%s' instead (arg#%d)",
				reflect.TypeOf(ch).String(), i+1))
		}

		elemType := reflect.TypeOf(ch).Elem()

		// Element type must not be repeated
		for j, e := range elemTypes {
			if e == elemType {
				panic(fmt.Sprintf(
					"All channels must have mutually different element types, arg#%d and arg#%d both have element type '%s'",
					j+1, i+1, e.String()))
			}
		}
		elemTypes[i] = elemType

		// Element type must be encodable with JSON
		switch elemType.Kind() {
		case reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
			panic(fmt.Sprintf(
				"Channel element type must be supported by JSON, got '%s' instead (arg#%d)",
				elemType.String(), i+1))
		case reflect.Map:
			if elemType.Key().Kind() != reflect.String {
				panic(fmt.Sprintf(
					"Channel element type must be supported by JSON, got '%s' instead (map keys must be 'string') (arg#%d)",
					elemType.String(), i+1))
			}
		}
	}
}
