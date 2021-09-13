package elevator

import (
	"encoding/json"
	"fmt"
	"net"

	"./conn"
	"./eventManager"
	"./utils"
)

// This struct holds received packet IDs for duplicate check.
// The struct holds three registers that are cycled through in order to prevent packets
// from being stored forever, while storing them for a sufficient amount of time
type recievedPackets struct {
	currentRegister int
	Packets         [3][]int
}

// Receiver function starts receiving data from network and starts routine to handle ack sending.
func Receiver() {
	var buf [1024]byte
	ackChan := make(chan int)
	conn := conn.DialBroadcastUDP(utils.CONNECTION_DATA_PORT)
	var rp recievedPackets

	go transmitAck(ackChan)

	for {
		// Reading data from buffer.
		n, _, e := conn.ReadFrom(buf[0:])

		if e != nil {
			fmt.Printf("bcast.Receiver(%d, ...):ReadFrom() failed: \"%+v\"\n", utils.CONNECTION_DATA_PORT, e)
		}
		var packet dataPacket
		json.Unmarshal(buf[0:n], &packet)

		if (packet.PacketID >> 8) != utils.ELEVATOR_ID {
			ackChan <- packet.PacketID
			if rp.handle(packet.PacketID) {
				// If packet is not already received or a loopback message send to Event Manager
				var p typeTaggedJSON
				json.Unmarshal(packet.D, &p)
				eventManager.PublishJSON(p.JSON, p.TypeId)
			}
		}
	}
}

// Thos loop creates and transmits ack packets for packet IDs received through channel
func transmitAck(ch <-chan int) {
	conn := conn.DialBroadcastUDP(utils.CONNECTION_ACK_PORT)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", utils.CONNECTION_ACK_PORT))
	for {
		packetID := <-ch
		d := ackPacket{ElevatorID: utils.ELEVATOR_ID, PacketID: packetID}
		jsonstr, err := json.Marshal(d)
		utils.CheckError(err)
		conn.WriteTo(jsonstr, addr)
	}
}

//handle method returns true if message is not already recieved and should be handled
func (p *recievedPackets) handle(packetID int) bool {
	if p.Packets[p.currentRegister] == nil {
		p.Packets[p.currentRegister] = make([]int, 0)
	}

	// if in current register
	for i := len(p.Packets[p.currentRegister]) - 1; i >= 0; i-- {
		if packetID == p.Packets[p.currentRegister][i] {
			return false

		}
	}

	// if in last register
	l := (p.currentRegister + 2) % 3
	for j := len(p.Packets[l]) - 1; j >= 0; j-- {
		if packetID == p.Packets[l][j] {
			return false
		}
	}
	// add to register
	p.Packets[p.currentRegister] = append(p.Packets[p.currentRegister], packetID)
	if len(p.Packets[p.currentRegister]) > utils.RX_PACKET_REGISTER_LENGTH {

		p.Packets[l] = nil
		p.currentRegister++
		p.currentRegister = p.currentRegister % 3
	}
	return true
}
