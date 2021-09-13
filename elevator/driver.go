package elevator

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"./eventManager"
	"./log"
	"./utils"
)

const _pollRate = 20 * time.Millisecond

var _mtx sync.Mutex
var _conn net.Conn

func init() {
	initTCP()
}

//Driver Module Function initializes the Driver module and start the go routines for
//polling buttons and sensor. The for-select cases are events from controller to set the behaviour
//of the elevator
func DriverModule() {

	log.PrintInf("Started")

	newOrderPub := make(chan NewOrderEvent)
	newCabOrderPub := make(chan NewCabOrderEvent)
	FloorUptPub := make(chan FloorUptEvent)
	obstructedPub := make(chan ObstructedEvent)

	ElevatorCtrSub := make(chan ElevatorCtrlEvent)
	OrderLampCtrSub := make(chan OrderLampCtrEvent)
	OrderLampsOffCtrSub := make(chan OrderLampsOffCtrEvent)

	eventManager.AddPublishers(newOrderPub, newCabOrderPub, newCabOrderPub, FloorUptPub, obstructedPub) //, stoppedPub)
	eventManager.AddSubscribers(ElevatorCtrSub, OrderLampCtrSub, OrderLampsOffCtrSub)

	_mtx = sync.Mutex{}

	go pollButtons(newOrderPub, newCabOrderPub)
	go pollFloorSensor(FloorUptPub)
	go pollObstructionSwitch(obstructedPub)

	for {
		select {
		case evt := <-ElevatorCtrSub:
			setElevatorControl(evt)
		case evt := <-OrderLampCtrSub:
			SetButtonLamp(evt.OrderType, evt.Floor, evt.IfTrue)
		case evt := <-OrderLampsOffCtrSub:
			TurnOffButtonLamps(evt.Floor, evt.AllButtons)
		}
	}
}

//Sets the control of the elevator accorting to ElevatorCrtlEvent
func setElevatorControl(evt ElevatorCtrlEvent) {
	floor := evt.Floor
	movement := evt.Movement
	behaviour := evt.Behaviour
	SetFloorIndicator(floor)
	SetMotorDirection(movement)
	if behaviour == behaviourDoorOpen {
		SetDoorOpenLamp(true)
	} else {
		SetDoorOpenLamp(false)
	}
}

func SetMotorDirection(dir Movement) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{1, byte(dir), 0, 0})
}

func SetButtonLamp(button OrderType, floor int, value bool) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{2, byte(button), byte(floor), toByte(value)})
}

func TurnOffButtonLamps(floor int, AllButtons bool) {
	if AllButtons {
		SetButtonLamp(orderHallUp, floor, false)
		SetButtonLamp(orderHallDown, floor, false)
		SetButtonLamp(orderCab, floor, false)
	} else {
		SetButtonLamp(orderHallUp, floor, false)
		SetButtonLamp(orderHallDown, floor, false)
	}

}
func SetFloorIndicator(floor int) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{3, byte(floor), 0, 0})
}

func SetDoorOpenLamp(value bool) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{4, toByte(value), 0, 0})
}

//The pollButtons function is modified to publish events when new hall and cab orders are pushed.
func pollButtons(newOrderPub chan<- NewOrderEvent, newCabOrderPub chan<- NewCabOrderEvent) {
	prev := make([][3]bool, utils.FLOOR_NUM)

	i := 0
	for {
		time.Sleep(_pollRate)

		for f := 0; f < utils.FLOOR_NUM; f++ {

			for b := OrderType(0); b < 3; b++ {
				v := getButton(b, f)

				if v != prev[f][b] && v {
					orderID := (utils.ELEVATOR_ID << 6) + (i & 63)
					i++
					if OrderType(b) == orderCab {
						evt := NewCabOrderEvent{utils.ELEVATOR_ID, f, orderID, orderCab}
						newCabOrderPub <- evt
					} else {
						evt := NewOrderEvent{utils.ELEVATOR_ID, f, orderID, OrderType(b)}
						newOrderPub <- evt
					}
				}
				prev[f][b] = v
			}
		}
	}
}

//pollFloorSensor is modified to publish floor update event when the elevator reaches a floor
func pollFloorSensor(floorUptPub chan<- FloorUptEvent) {
	prev := -1
	for {
		time.Sleep(_pollRate)
		v := getFloor()
		if v != prev && v != -1 {
			evt := FloorUptEvent{utils.ELEVATOR_ID, v}
			floorUptPub <- evt
		}
		prev = v
	}
}

//pollObstructionSwitch is modified to publish an obstructedEvent when turned off and on
func pollObstructionSwitch(obstructedPub chan<- ObstructedEvent) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := getObstruction()
		if v != prev {
			evt := ObstructedEvent{utils.ELEVATOR_ID, v}
			obstructedPub <- evt
		}
		prev = v
	}
}

func initTCP() {
	var err error
	fmt.Println("Elevator port" + strconv.Itoa(utils.ELEVATOR_PORT))
	_conn, err = net.Dial("tcp", "localhost:"+strconv.Itoa(utils.ELEVATOR_PORT))
	utils.CheckError(err)
}

func getButton(button OrderType, floor int) bool {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{6, byte(button), byte(floor), 0})
	var buf [4]byte
	_conn.Read(buf[:])
	return toBool(buf[1])
}

func getFloor() int {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{7, 0, 0, 0})
	var buf [4]byte
	_conn.Read(buf[:])
	if buf[1] != 0 {
		return int(buf[2])
	} else {
		return -1
	}
}

func getObstruction() bool {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{9, 0, 0, 0})
	var buf [4]byte
	_conn.Read(buf[:])
	return toBool(buf[1])
}

func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	var b bool = false
	if a != 0 {
		b = true
	}
	return b
}
