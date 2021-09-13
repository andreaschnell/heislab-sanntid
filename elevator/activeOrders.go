package elevator

import (
	"./eventManager"
	"./log"
	"./utils"
)

//List with HallUp/HallDown orders for each floor
type HallOrders struct {
	Orders [utils.FLOOR_NUM][utils.ORDER_TYPE_NUM - 1]int
}

//Map over hall orders for the different elevators, sorted by elevator ID
var HallOrdersMap map[int]HallOrders

//The Queue modules keeps track on all the elevators Hall Orders
func ActiveOrdersModule() {
	log.PrintDbg("Started")

	activeOrdersAnsPub := make(chan ActiveOrdersAnsEvent)

	activeOrdersReqSub := make(chan ActiveOrdersReqEvent)
	assignedSub := make(chan AssignedEvent)
	orderCompleteSub := make(chan OrderCompleteEvent)
	availabilitySub := make(chan AvailabilityEvent)
	unavailableOrdersHandledSub := make(chan UnavailableOrdersHandledEvent)

	eventManager.AddPublishers(activeOrdersAnsPub)
	eventManager.AddSubscribers(activeOrdersReqSub, assignedSub, orderCompleteSub, availabilitySub, unavailableOrdersHandledSub)

	HallOrdersMap = make(map[int]HallOrders)

	for {
		select {
		case evt := <-assignedSub:
			AddHallOrders(evt)
		case evt := <-orderCompleteSub:
			RemoveFloorHallOrders(evt.Floor)
		case evt := <-activeOrdersReqSub:
			hallOrders := HallOrdersMap[evt.ElevatorID]
			ActiveOrders := ActiveOrdersAnsEvent{evt.ElevatorID, hallOrders.Orders}
			activeOrdersAnsPub <- ActiveOrders
		case evt := <-availabilitySub:
			if !singleElevatorAvailable() {
				if !evt.Availabable && evt.ElevatorID == utils.ELEVATOR_ID {
					deleteAllHallOrders(evt.ElevatorID)
					log.PrintDbg("Deleted hall orders for elev", utils.ELEVATOR_ID, " is ", HallOrdersMap[utils.ELEVATOR_ID])
				}
			}
		case evt := <-unavailableOrdersHandledSub:
			if evt.Handled {
				deleteAllHallOrders(evt.ElevatorID)
			}
		}
	}
}

//Deletes all HallOrders from an elevator if its gets unavailable/disconnected
func deleteAllHallOrders(elevatorID int) {
	for floor := 0; floor < utils.FLOOR_NUM; floor++ {
		for orderType := 0; orderType < utils.ORDER_TYPE_NUM-1; orderType++ {
			elevatorOrders := HallOrdersMap[elevatorID]
			elevatorOrders.Orders[floor][orderType] = 0
			HallOrdersMap[elevatorID] = elevatorOrders
		}
	}
}

//Adds Hall orders for each elevator in HallOrderMap
func AddHallOrders(assignedEvent AssignedEvent) {
	log.PrintDbg("Assigned elev to add", assignedEvent.ElevatorID)
	ActiveOrders := HallOrdersMap[assignedEvent.ElevatorID]
	ActiveOrders.Orders[assignedEvent.Floor][assignedEvent.OrderType] = 1
	HallOrdersMap[assignedEvent.ElevatorID] = ActiveOrders
	log.PrintDbg("Assigned hall orders for elev", utils.ELEVATOR_ID, " is HallOrdersMap[utils.ELEVATOR_ID]")
}

//Removes completed orders for each elevator in HallOrderMap
func RemoveHallOrders(ElevatorID int, floor int) {
	ActiveOrders := HallOrdersMap[ElevatorID]
	for i := 0; i < utils.ORDER_TYPE_NUM-1; i++ {
		ActiveOrders.Orders[floor][i] = 0
	}
	HallOrdersMap[ElevatorID] = ActiveOrders
}
func RemoveFloorHallOrders(floor int) {
	for i := 0; i < utils.ELEVATOR_MAX_NUM; i++ {
		RemoveHallOrders(i, floor)
	}
}
