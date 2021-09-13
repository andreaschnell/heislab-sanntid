package elevator

import (
	"sync"
	"time"

	"./eventManager"
	"./log"
	"./utils"
)

//Struct for each order
type Order struct {
	ID       int
	elevsReg []bool
	costs    []int
}

//Struct to be used by AssignedElevator map, contain each chosen ElevID from all elevators
//and if all elevators are registerd for recent orderIDs
type AssignedElevatorIDs struct {
	AssignedElevsList []int
	elevsReg          []bool
}

//Map of the active orders sorted by order ID
var orderMap map[int]Order

//Slice of the elevator statuses on availability index = elevator ID
var elevatorStatus []bool

//Map with Key = OrderID and maps to the AssignedElevatorIDs struct
var AssignedElevators map[int]AssignedElevatorIDs

//Assigner Module function recieves CostResultEvents from all elevators, Assignes the order
//to the Elev with lowest cost and send the order to Controller if all
//Elevators returns the same assigned elev
func AssignerModule() {
	log.PrintInf("Started")

	assignedPub := make(chan AssignedEvent)
	activeOrdersReqPub := make(chan ActiveOrdersReqEvent)
	newOrderPub := make(chan NewOrderEvent)
	unavailableOrdersHandledPub := make(chan UnavailableOrdersHandledEvent)
	checkAssignedElevPub := make(chan CheckAssignedElevEvent)
	assignedOKPub := make(chan AssignedOKEvent)

	costResultSub := make(chan CostResultEvent)
	availabilitySub := make(chan AvailabilityEvent)
	connectSub := make(chan ConnectionEvent)
	activeOrdersAnsSub := make(chan ActiveOrdersAnsEvent)
	assignedOKSub := make(chan AssignedOKEvent)
	checkAssignedElevSub := make(chan CheckAssignedElevEvent)

	eventManager.AddPublishers(assignedPub, newOrderPub, activeOrdersReqPub, unavailableOrdersHandledPub, checkAssignedElevPub, assignedOKPub)
	eventManager.AddSubscribers(costResultSub, availabilitySub, activeOrdersAnsSub, connectSub, assignedOKSub, checkAssignedElevSub)

	i := 0
	go distributeOrders(&i, activeOrdersAnsSub, newOrderPub, unavailableOrdersHandledPub)

	elevatorStatus = make([]bool, utils.ELEVATOR_MAX_NUM)
	elevatorStatus[utils.ELEVATOR_ID] = true
	orderMap = make(map[int]Order)
	AssignedElevators = make(map[int]AssignedElevatorIDs)
	mtx := &sync.Mutex{}
	for {
		select {
		case evt := <-costResultSub:
			readyToServe := registerOrder(evt.ElevatorID, evt.OrderID, evt.Score)
			var elevToServe int
			if readyToServe {
				order := orderMap[evt.OrderID]
				elevToServe = assignElevToServeOrder(order.costs)
				checkAssignedElev := CheckAssignedElevEvent{utils.ELEVATOR_ID, elevToServe, order.ID, evt.Floor, evt.OrderType}
				if elevatorStatus[utils.ELEVATOR_ID] {
					checkAssignedElevPub <- checkAssignedElev
				}
				go checkForSameResult(checkAssignedElevSub, assignedOKPub, checkAssignedElev, mtx)
			}
		case evt := <-assignedOKSub:
			singleMode := false
			if len(ReturnActiveElevatorsID()) == 1 {
				singleMode = true
			}
			if evt.SameAssigned {
				assignedEvent := AssignedEvent{evt.AssignedElevatorID, evt.OrderID, evt.Floor, evt.OrderType, singleMode}
				assignedPub <- assignedEvent
				delete(orderMap, evt.OrderID)
			} else {
				ActiveElevs := ReturnActiveElevatorsID()
				for _, v := range ActiveElevs {
					assignedEvent := AssignedEvent{v, evt.OrderID, evt.Floor, evt.OrderType, singleMode}
					assignedPub <- assignedEvent
					delete(orderMap, evt.OrderID)
				}
			}
		case evt := <-availabilitySub:
			if !singleElevatorAvailable() {
				elevatorStatus[evt.ElevatorID] = evt.Availabable
				if (evt.ElevatorID != utils.ELEVATOR_ID) && !evt.Availabable {
					ActiveElevatorID := ReturnActiveElevatorsID()
					if len(ActiveElevatorID) != 0 && ActiveElevatorID[0] == utils.ELEVATOR_ID {
						activeOrdersReqPub <- ActiveOrdersReqEvent{evt.ElevatorID}
					}
				}
			}
			if evt.ElevatorID != utils.ELEVATOR_ID {
				elevatorStatus[evt.ElevatorID] = evt.Availabable
			}
		case evt := <-connectSub:
			elevatorStatus[evt.ElevatorID] = evt.Connect
			if !evt.Connect {
				ActiveElevatorID := ReturnActiveElevatorsID()
				if len(ActiveElevatorID) != 0 && ActiveElevatorID[0] == utils.ELEVATOR_ID {
					activeOrdersReqPub <- ActiveOrdersReqEvent{evt.ElevatorID}
				}
			}
		}
	}
}

func singleElevatorAvailable() bool {
	activeElevs := ReturnActiveElevatorsID()
	if len(activeElevs) == 1 && activeElevs[0] == utils.ELEVATOR_ID {
		return true
	}
	return false
}

func generateOrderID(i *int) int {
	ival := *i
	orderID := ((utils.ELEVATOR_MAX_NUM + utils.ELEVATOR_ID) << 6) + (ival & 63)
	*i = ival + 1
	return orderID
}

//Generates and publishes new orders whenever a new active order set is received on the subscribed event channel
func distributeOrders(i *int, activeOrdersAnsSub chan ActiveOrdersAnsEvent, newOrderPub chan NewOrderEvent, unavailableOrdersHandledPub chan UnavailableOrdersHandledEvent) {
	for evt := range activeOrdersAnsSub {
		for floor := 0; floor < utils.FLOOR_NUM; floor++ {
			for orderType, v := range evt.ActiveOrders[floor] {
				if v == 1 {
					orderID := generateOrderID(i)
					NewOrder := NewOrderEvent{utils.ELEVATOR_ID, floor, orderID, OrderType(orderType)}
					newOrderPub <- NewOrder
				}
			}
		}
		Handled := UnavailableOrdersHandledEvent{evt.ElevatorID, true}
		unavailableOrdersHandledPub <- Handled
	}
}

//Registrers a new order in the global orderMap variable, with order ID as key. Returns wheather the order is ready to serveor not
func registerOrder(elevatorID int, orderID int, cost int) bool {
	var readyToServe bool
	_, exist := orderMap[orderID]
	var newOrder Order
	if exist {
		newOrder = orderMap[orderID]
		newOrder.elevsReg[elevatorID] = true
		newOrder.costs[elevatorID] = cost

	} else {
		costs := make([]int, utils.ELEVATOR_MAX_NUM)
		elevsReg := make([]bool, utils.ELEVATOR_MAX_NUM)
		elevsReg[elevatorID] = true
		costs[elevatorID] = cost
		newOrder = Order{orderID, elevsReg, costs}
	}
	for i, v := range newOrder.elevsReg {
		if v == elevatorStatus[i] {
			readyToServe = true
		} else {
			readyToServe = false
			break
		}
	}
	orderMap[orderID] = newOrder
	return readyToServe
}

//Finds elevator with the lowest cost and assignes it to that elevator. Prioritizes smalles elevator ID
func assignElevToServeOrder(cost []int) int {
	activeElevatorsID := ReturnActiveElevatorsID()
	elevId := activeElevatorsID[0]
	costVal := cost[elevId]
	for _, v := range activeElevatorsID {
		if cost[v] < costVal {
			costVal = cost[v]
			elevId = v
		}
	}
	return elevId
}

func ReturnActiveElevatorsID() []int {
	var activeElevatorsID []int
	for i, v := range elevatorStatus {
		if v {
			activeElevatorsID = append(activeElevatorsID, i)
		}
	}
	return activeElevatorsID
}

//Run as goroutine to compare the assigned elevators from each of the active elevator
//Returns true/false if AssignedElevIDs are/are not the same
//and false if timer runs out.
func checkForSameResult(CheckAssignedElevSub chan CheckAssignedElevEvent, assignedOKPub chan AssignedOKEvent, assigned CheckAssignedElevEvent, mtx *sync.Mutex) {

	AssignedTimer := time.NewTimer(utils.MAX_DECIDE_TIME * time.Millisecond)

	for {
		select {
		case evt := <-CheckAssignedElevSub:
			if SameAssignedElevs(evt, mtx) {
				AssignedTimer.Stop()
				AssignedOK := AssignedOKEvent{true, evt.AssignedElevatorID, evt.OrderID, evt.Floor, evt.OrderType}
				assignedOKPub <- AssignedOK
				mtx.Lock()
				delete(AssignedElevators, evt.OrderID)
				mtx.Unlock()
				return
			} else {
				activeElevatorsID := ReturnActiveElevatorsID()
				NumActiveElevators := len(activeElevatorsID)
				ElevsRegTrue := 0
				mtx.Lock()
				elevsReg := AssignedElevators[evt.OrderID].elevsReg
				mtx.Unlock()
				for i := 0; i < len(elevsReg); i++ {
					if elevsReg[i] {
						ElevsRegTrue++
					}
				}
				if ElevsRegTrue == NumActiveElevators {
					AssignedOK := AssignedOKEvent{false, evt.AssignedElevatorID, evt.OrderID, evt.Floor, evt.OrderType}
					assignedOKPub <- AssignedOK
					mtx.Lock()
					delete(AssignedElevators, evt.OrderID)
					mtx.Unlock()
					AssignedTimer.Stop()
					return
				}
			}
		case <-AssignedTimer.C:
			AssignedOK := AssignedOKEvent{false, assigned.AssignedElevatorID, assigned.OrderID, assigned.Floor, assigned.OrderType}
			assignedOKPub <- AssignedOK
			mtx.Lock()
			delete(AssignedElevators, assigned.OrderID)
			mtx.Unlock()
			return
		}
	}
}

//Checks if all Elevators returned the same Elevator to serve.
//Adds the AssignedElevatorIDs to the corresponing OrderID in the map AssignedElevators.
//Returns true if all Active Elevators agrees
func SameAssignedElevs(AssignedElev CheckAssignedElevEvent, mtx *sync.Mutex) bool {
	var AllElevsRegistered bool
	var sameAssignedElevs bool
	activeElevatorsID := ReturnActiveElevatorsID()
	mtx.Lock()
	_, exist := AssignedElevators[AssignedElev.OrderID]
	mtx.Unlock()
	var AssignedElevs AssignedElevatorIDs
	if exist {
		mtx.Lock()
		AssignedElevs = AssignedElevators[AssignedElev.OrderID]
		mtx.Unlock()
		AssignedElevs.elevsReg[AssignedElev.ElevatorID] = true
		AssignedElevs.AssignedElevsList[AssignedElev.ElevatorID] = AssignedElev.AssignedElevatorID
	} else {
		elevsReg := make([]bool, utils.ELEVATOR_MAX_NUM)
		AssignedElevsList := make([]int, utils.ELEVATOR_MAX_NUM)
		elevsReg[AssignedElev.ElevatorID] = true
		AssignedElevsList[AssignedElev.ElevatorID] = AssignedElev.AssignedElevatorID
		AssignedElevs = AssignedElevatorIDs{AssignedElevsList, elevsReg}
	}
	for _, v := range activeElevatorsID {
		if AssignedElevs.elevsReg[v] {
			AllElevsRegistered = true
		} else {
			AllElevsRegistered = false
			break
		}
	}
	if AllElevsRegistered {
		sameAssignedElevs = true
		if len(activeElevatorsID) > 1 {
			for i := 0; i < len(activeElevatorsID)-1; i++ {
				if AssignedElevs.AssignedElevsList[activeElevatorsID[i]] != AssignedElevs.AssignedElevsList[activeElevatorsID[i+1]] {
					sameAssignedElevs = false
				}
			}
		}
	}
	mtx.Lock()
	AssignedElevators[AssignedElev.OrderID] = AssignedElevs
	mtx.Unlock()
	return sameAssignedElevs
}

func InitTimer() *time.Timer {
	timer := time.NewTimer(3 * time.Second)
	timer.Stop()
	return timer
}
