package elevator

import (
	"./utils"
)

//OrderCompleteEvent I used to signal when an Order is completed
type OrderCompleteEvent struct {
	ElevatorID int
	Floor      int
}

//ElevatorControlEvent is used to signal elevator control commands
type ElevatorCtrlEvent struct {
	Floor     int
	Behaviour ElevatorBehaviour
	Movement  Movement
}

//CostResultEvent is used to signal a result of a cost calculation
type CostResultEvent struct {
	ElevatorID int
	OrderID    int
	Score      int
	Floor      int
	OrderType  OrderType
}

//FloorUptEvent happens everytime elevator reaches a new floor
type FloorUptEvent struct {
	ElevatorID int
	Floor      int
}

type OrderType int

const (
	orderHallUp OrderType = iota
	orderHallDown
	orderCab
)

//NewOrderEvent happens everytime there is a new panel order
type NewOrderEvent struct {
	ElevatorID int
	Floor      int
	OrderID    int
	OrderType  OrderType
}

//NewCabOrderEvent happens everytime there is a new cab order
type NewCabOrderEvent struct {
	ElevatorID int
	Floor      int
	OrderID    int
	OrderType  OrderType
}

//ObstructedEvent happens everytime the elevator is obstructed or the obstruction goes away
type ObstructedEvent struct {
	ElevatorID int
	Obstructed bool
}

//Assigned Event is used to signal that an eevator has been selected for an order
type AssignedEvent struct {
	ElevatorID int
	OrderID    int
	Floor      int
	OrderType  OrderType
	SingleMode bool
}

//CheckAssignedElevEvent is used to send the assigned elevator to the other elevator for comparison
type CheckAssignedElevEvent struct {
	ElevatorID         int
	AssignedElevatorID int
	OrderID            int
	Floor              int
	OrderType          OrderType
}

//AssignedOKEvent is used to check if the assigned elevator is the same for all elevators
type AssignedOKEvent struct {
	SameAssigned       bool
	AssignedElevatorID int
	OrderID            int
	Floor              int
	OrderType          OrderType
}

//Elevator Availability event used to signal that the availability of an elevator has changed
type AvailabilityEvent struct {
	ElevatorID  int
	Availabable bool
}

//ConnectionEvent is used to signal if an elevator is connected or not
type ConnectionEvent struct {
	ElevatorID int
	Connect    bool
}

//OrderLampCtrEvent is used to signal if the cab order lights or hall order lighst should turn on
type OrderLampCtrEvent struct {
	Floor     int
	OrderType OrderType
	IfTrue    bool
}

//OrderLampsOffCtrEvent is used to turn off either hall buttons of all buttons
type OrderLampsOffCtrEvent struct {
	Floor      int
	AllButtons bool
}

//ActiveORdersReqEvent is used to request one elevators active hall orders
type ActiveOrdersReqEvent struct {
	ElevatorID int
}

//ActiveOrderAnsEvent return the requested elevators active orders
type ActiveOrdersAnsEvent struct {
	ElevatorID   int
	ActiveOrders [utils.FLOOR_NUM][utils.ORDER_TYPE_NUM - 1]int
}

//UnavailableOrdersHandledEvent is used to signal that the unavailable elevators hall orders
//are handled by the other elevators
type UnavailableOrdersHandledEvent struct {
	ElevatorID int
	Handled    bool
}
