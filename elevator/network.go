package elevator

import (
	"reflect"
	"time"

	"./eventManager"
	"./log"
	"./utils"
)

type typeTaggedJSON struct {
	TypeId string
	JSON   []byte
}

type dataPacket struct {
	PacketID int
	D        []byte
}

// Network module function.
func NetworkModule() {

	log.PrintInf("Started")

	connectPub := make(chan ConnectionEvent)

	connectSub := make(chan ConnectionEvent)
	newOrderSub := make(chan NewOrderEvent)
	costResultSub := make(chan CostResultEvent)
	availabilitySub := make(chan AvailabilityEvent)
	orderCompleteSub := make(chan OrderCompleteEvent)
	checkAssignedElevSub := make(chan CheckAssignedElevEvent)

	eventManager.AddPublishers(connectPub)
	eventManager.AddSubscribers(connectSub, newOrderSub, costResultSub, availabilitySub, orderCompleteSub, checkAssignedElevSub)

	// Start transmitting and receiving as well as connection checking.
	// Subscriber channels from eventmanager is fed directly to the transmitter.
	// Received events i also sent directly to the event manager.
	go Transmitter(filterElevatorID, connectSub, newOrderSub, costResultSub, availabilitySub, orderCompleteSub, checkAssignedElevSub)
	go Receiver()
	go ConnectionCheck(connectPub)

	for {
		time.Sleep(time.Second)
	}
}

// FIlter function for transmitting events. This ensures that only events
// from this module is sent over network and no feedback of packet will occur.
func filterElevatorID(v reflect.Value) bool {
	if v.Field(0).Kind() == reflect.Int {
		if int(v.Field(0).Int()) == utils.ELEVATOR_ID {
			return true
		}
	}
	return false
}
