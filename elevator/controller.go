package elevator

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"

	"./eventManager"
	"./log"
	"./utils"
)

//Type definition of the different elevator behaviours
type ElevatorBehaviour int

const (
	behaviourIdle ElevatorBehaviour = iota
	behaviourDoorOpen
	behaviourMoving
	behaviourObstructed
)

//Type definition elevator movement
type Movement int

const (
	moveDown Movement = -1 + iota
	moveStop
	moveUp
)

//Structure of the Elevator state with different state variables
type ElevatorState struct {
	ElevatorID   int
	Floor        int
	Behaviour    ElevatorBehaviour
	Movement     Movement
	Available    bool
	ActiveOrders [utils.FLOOR_NUM][utils.ORDER_TYPE_NUM]int
}

//Global declaration of the elevator state of this program
var elevatorState ElevatorState

//Timers used in the controller module
var obstructTimer *time.Timer
var doorTimer *time.Timer
var inBetweenFloorTimer *time.Timer

//ControllerModule function
func ControllerModule() {

	elevatorState.ElevatorID = utils.ELEVATOR_ID
	elevatorState.Available = true

	log.PrintInf("Started")

	orderCompletePub := make(chan OrderCompleteEvent)
	elevatorCtrlPub := make(chan ElevatorCtrlEvent)
	costResultPub := make(chan CostResultEvent)
	availabilityPub := make(chan AvailabilityEvent)
	OrderLampCtrPub := make(chan OrderLampCtrEvent)
	OrderLampsOffCtrPub := make(chan OrderLampsOffCtrEvent)

	orderCompleteSub := make(chan OrderCompleteEvent)
	floorUptSub := make(chan FloorUptEvent)
	newOrderSub := make(chan NewOrderEvent)
	obstructedSub := make(chan ObstructedEvent)
	newCabOrderSub := make(chan NewCabOrderEvent)
	assignedSub := make(chan AssignedEvent)

	eventManager.AddPublishers(orderCompletePub, elevatorCtrlPub, costResultPub, availabilityPub, OrderLampCtrPub, OrderLampsOffCtrPub)
	eventManager.AddSubscribers(orderCompleteSub, floorUptSub, newOrderSub, obstructedSub, newCabOrderSub, assignedSub)

	doorTimer = timerInit()
	obstructTimer = timerInit()
	inBetweenFloorTimer = timerInit()

	backupFileName := "cab_orders_backup" + strconv.Itoa(elevatorState.ElevatorID)
	orders := getBackupedCabOrders(backupFileName)
	addBackupedCaborders(&elevatorState, orders)
	backupFile, err := os.Create(backupFileName)
	utils.CheckError(err)

	initLamps()
	for {
		select {
		case evt := <-floorUptSub:
			newFloor := evt.Floor
			d := elevatorControlOnFloorUpt(newFloor)
			elevatorCtrlPub <- d
			if d.Movement == moveStop {
				inBetweenFloorTimer.Stop()
				OrderComplete := OrderCompleteEvent{utils.ELEVATOR_ID, evt.Floor}
				orderCompletePub <- OrderComplete
				backupCabOrders(backupFile, elevatorState.ActiveOrders)
			} else if elevatorState.Available {
				resetTimer(inBetweenFloorTimer, utils.MAX_TRAVEL_TIME)

			}
		case evt := <-orderCompleteSub:
			var AllButtons bool
			if evt.ElevatorID == utils.ELEVATOR_ID {
				AllButtons = true
			} else {
				AllButtons = false
			}
			l := OrderLampsOffCtrEvent{evt.Floor, AllButtons}
			OrderLampsOffCtrPub <- l
			for i := 0; i < utils.ORDER_TYPE_NUM-1; i++ {
				elevatorState.ActiveOrders[evt.Floor][i] = 0
			}

		case evt := <-newOrderSub:
			cost := TimeToServeOrder(elevatorState, evt.OrderType, evt.Floor)
			d := CostResultEvent{utils.ELEVATOR_ID, evt.OrderID, cost, evt.Floor, evt.OrderType}
			if elevatorState.Available {
				costResultPub <- d
			}
		case <-doorTimer.C:
			d := ElevatorCtrlEvent{}
			d.Floor = elevatorState.Floor
			if elevatorState.Behaviour == behaviourObstructed {
				d.Movement = moveStop
				d.Behaviour = behaviourDoorOpen
			} else {
				chooseDirUptState()
				d.Movement = elevatorState.Movement
				d.Behaviour = elevatorState.Behaviour
				if d.Movement != moveStop && elevatorState.Available {
					resetTimer(inBetweenFloorTimer, utils.MAX_TRAVEL_TIME)
				}
			}
			elevatorCtrlPub <- d
		case evt := <-obstructedSub:

			d := ElevatorCtrlEvent{}
			d.Floor = elevatorState.Floor
			if evt.Obstructed && elevatorState.Behaviour == behaviourDoorOpen {
				resetTimer(obstructTimer, utils.MAX_OBSTRUCT_TIME)
				elevatorState.Behaviour = behaviourObstructed
				d.Movement = moveStop
				d.Behaviour = behaviourDoorOpen
				elevatorCtrlPub <- d
			} else if !evt.Obstructed && elevatorState.Behaviour == behaviourObstructed {
				obstructTimer.Stop()
				if !elevatorState.Available {
					elevatorState.Available = true
					d := AvailabilityEvent{elevatorState.ElevatorID, true}
					availabilityPub <- d
				}
				elevatorState.Available = true
				chooseDirUptState()
				d.Behaviour = elevatorState.Behaviour
				d.Movement = elevatorState.Movement
				if d.Movement != moveStop {
					resetTimer(inBetweenFloorTimer, utils.MAX_TRAVEL_TIME)
				} else {
					l := OrderLampsOffCtrEvent{elevatorState.Floor, true}
					OrderLampsOffCtrPub <- l
				}
				elevatorCtrlPub <- d
			}
		case <-obstructTimer.C:
			elevatorState.Available = false
			d := AvailabilityEvent{elevatorState.ElevatorID, elevatorState.Available}
			availabilityPub <- d
			deleteHallOrders()
		case evt := <-newCabOrderSub:
			newOrderController(evt.Floor, evt.OrderType, OrderLampsOffCtrPub, OrderLampCtrPub, elevatorCtrlPub, orderCompletePub)
			l := OrderLampCtrEvent{evt.Floor, evt.OrderType, true}
			OrderLampCtrPub <- l
			backupCabOrders(backupFile, elevatorState.ActiveOrders)
		case evt := <-assignedSub:
			if !evt.SingleMode {
				l := OrderLampCtrEvent{evt.Floor, evt.OrderType, true}
				OrderLampCtrPub <- l
			}
			if evt.ElevatorID == elevatorState.ElevatorID {
				newOrderController(evt.Floor, evt.OrderType, OrderLampsOffCtrPub, OrderLampCtrPub, elevatorCtrlPub, orderCompletePub)
			}
		case <-inBetweenFloorTimer.C:
			go onMotorStop(availabilityPub, elevatorCtrlPub)
		}
	}
}

func initLamps() {
	for i := 0; i < utils.FLOOR_NUM; i++ {
		TurnOffButtonLamps(i, false)
	}
}

func resetTimer(timer *time.Timer, sec int) {
	timer.Stop()
	timer.Reset(time.Duration(sec) * time.Second)
}

func addBackupedCaborders(elevator *ElevatorState, orders [utils.FLOOR_NUM]int) {
	for floor, order := range orders {
		elevator.ActiveOrders[floor][orderCab] = order
	}
}

func getBackupedCabOrders(filename string) [utils.FLOOR_NUM]int {
	file := openFile(filename)
	file.Seek(0, 0)
	scanner := bufio.NewScanner(file)
	var orders [utils.FLOOR_NUM]int
	i := 0
	for scanner.Scan() {
		num, err := strconv.Atoi(scanner.Text())
		utils.CheckError(err)
		orders[i] = num
		i++
	}
	defer file.Close()
	return orders
}

func backupCabOrders(file *os.File, orders [utils.FLOOR_NUM][utils.ORDER_TYPE_NUM]int) {
	file.Seek(0, 0)
	file.Truncate(0)
	for floor := range orders {
		_, err := file.WriteString(fmt.Sprintf("%d\n", orders[floor][orderCab]))
		utils.CheckError(err)
	}
}

func openFile(filename string) *os.File {
	var file *os.File
	var err error
	if fileExists(filename) {
		file, err = os.OpenFile(filename, os.O_RDWR, 0644)
		utils.CheckError(err)

	} else {
		file, err = os.Create(filename)
		utils.CheckError(err)
	}
	return file
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

//Constantly tries to turn on motor until the floor has been updated, meaning the motor has started working again
func onMotorStop(availabilityPub chan AvailabilityEvent, elevatorCtrlPub chan ElevatorCtrlEvent) {
	elevatorState.Available = false
	d := AvailabilityEvent{utils.ELEVATOR_ID, elevatorState.Available}

	availabilityPub <- d
	deleteHallOrders()
	floor := elevatorState.Floor
	var dir Movement
	if elevatorState.Movement == moveStop {
		if floor == 0 {
			dir = moveUp
		} else {
			dir = moveDown
		}
	} else {
		dir = elevatorState.Movement
	}
	c := ElevatorCtrlEvent{floor, behaviourMoving, dir}
	for {
		if floor != elevatorState.Floor {
			break
		}
		elevatorCtrlPub <- c
		time.Sleep(1 * time.Second)
	}
	elevatorState.Available = true
	a := AvailabilityEvent{elevatorState.ElevatorID, elevatorState.Available}
	availabilityPub <- a
}

func timerInit() *time.Timer {
	timer := time.NewTimer(3 * time.Second)
	timer.Stop()
	return timer
}

func deleteHallOrders() {
	for floor := 0; floor < utils.FLOOR_NUM; floor++ {
		for orderType := 0; orderType < utils.ORDER_TYPE_NUM-1; orderType++ {
			elevatorState.ActiveOrders[floor][orderType] = 0
		}
	}
}

//Decides and sets the elevator control based on the new order event and the current state of the elevator
func newOrderController(floor int, orderType OrderType,
	OrderLampsOffCtrPub chan OrderLampsOffCtrEvent,
	OrderLampCtrPub chan OrderLampCtrEvent,
	elevatorCtrlPub chan ElevatorCtrlEvent,
	orderCompletePub chan OrderCompleteEvent) {
	elevatorState.ActiveOrders[floor][orderType] = 1
	switch elevatorState.Behaviour {
	case behaviourMoving:
		chooseDirUptState()
		d, ButtonLamp := elevatorCtrFromOrder(floor)
		elevatorCtrlPub <- d
		if ButtonLamp {
			l := OrderLampsOffCtrEvent{floor, false}
			OrderLampsOffCtrPub <- l
		}
	case behaviourIdle:
		d, ButtonLamp := elevatorCtrFromOrder(floor)
		elevatorCtrlPub <- d
		if elevatorState.Available && d.Movement != moveStop {
			resetTimer(inBetweenFloorTimer, utils.MAX_TRAVEL_TIME)

		}
		if ButtonLamp {
			AllButtons := true
			l := OrderLampsOffCtrEvent{floor, AllButtons}
			OrderLampsOffCtrPub <- l
			OrderComplete := OrderCompleteEvent{utils.ELEVATOR_ID, floor}
			orderCompletePub <- OrderComplete
		}
	case behaviourObstructed:
		if floor == elevatorState.Floor {
			clearOrderOnCurrentFloor(&elevatorState)
			AllButtons := true
			l := OrderLampsOffCtrEvent{floor, AllButtons}
			OrderLampsOffCtrPub <- l
			OrderComplete := OrderCompleteEvent{utils.ELEVATOR_ID, floor}
			orderCompletePub <- OrderComplete
		}
	case behaviourDoorOpen:
		if floor == elevatorState.Floor {
			doorTimer.Reset(utils.DOOR_OPEN_TIME * time.Second)
			clearOrderOnCurrentFloor(&elevatorState)
			AllButtons := true
			l := OrderLampsOffCtrEvent{floor, AllButtons}
			OrderLampsOffCtrPub <- l
			OrderComplete := OrderCompleteEvent{utils.ELEVATOR_ID, floor}
			orderCompletePub <- OrderComplete
		}
	}
}

func clearOrderOnCurrentFloor(state *ElevatorState) {
	for i := 0; i < utils.ORDER_TYPE_NUM; i++ {
		state.ActiveOrders[state.Floor][i] = 0
	}
}

//Decides how the elevator should be controlled based on the new floor and the state of the elevator
func elevatorControlOnFloorUpt(newFloor int) ElevatorCtrlEvent {
	elevatorState.Floor = newFloor
	if Requests_shouldStop(elevatorState) == 1 {
		elevatorState.Behaviour = behaviourDoorOpen
		clearOrderOnCurrentFloor(&elevatorState)
		elevatorCtr := ElevatorCtrlEvent{newFloor, behaviourDoorOpen, moveStop}
		resetTimer(doorTimer, utils.DOOR_OPEN_TIME)
		return elevatorCtr
	} else if newFloor == utils.FLOOR_NUM-1 || newFloor == 0 {
		elevatorCtr := ElevatorCtrlEvent{newFloor, behaviourIdle, moveStop}
		return elevatorCtr
	} else {
		elevatorCtr := ElevatorCtrlEvent{newFloor, elevatorState.Behaviour, elevatorState.Movement}
		return elevatorCtr
	}
}

//Calculates the elevator control event that should happen based on the current active requests, elevator state and
//if the order lights should be turned off
func elevatorCtrFromOrder(FloorRequest int) (ElevatorCtrlEvent, bool) {
	var elevatorCtr ElevatorCtrlEvent
	var ButtonLampsOff bool
	if (elevatorState.Behaviour == behaviourDoorOpen || elevatorState.Behaviour == behaviourIdle) && elevatorState.Floor == FloorRequest {
		elevatorState.Behaviour = behaviourDoorOpen
		clearOrderOnCurrentFloor(&elevatorState)
		elevatorCtr = ElevatorCtrlEvent{FloorRequest, behaviourDoorOpen, moveStop}
		doorTimer.Reset(utils.DOOR_OPEN_TIME * time.Second)
		ButtonLampsOff = true
	} else if elevatorState.Behaviour == behaviourIdle {
		chooseDirUptState()
		elevatorCtr = ElevatorCtrlEvent{elevatorState.Floor, elevatorState.Behaviour, elevatorState.Movement}
	} else {
		elevatorCtr = ElevatorCtrlEvent{elevatorState.Floor, elevatorState.Behaviour, elevatorState.Movement}
		ButtonLampsOff = false
	}
	return elevatorCtr, ButtonLampsOff
}

//Updates the elevatorState movement and behaviour based on the current active orders
func chooseDirUptState() {
	elevatorState.Movement = Requests_chooseDirection(elevatorState)
	if elevatorState.Movement == moveStop {
		elevatorState.Behaviour = behaviourIdle
	} else {
		elevatorState.Behaviour = behaviourMoving
	}
}

func StartElevator() {
	SetMotorDirection(moveUp)
	elevatorState.Behaviour = behaviourMoving
	elevatorState.Movement = moveUp
}
