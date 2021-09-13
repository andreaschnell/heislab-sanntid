package elevator

import (
	"./utils"
)

func TimeToServeOrder(state ElevatorState, b OrderType, f int) int {
	e := state
	e.ActiveOrders[f][b] = 1

	var arrivedAtOrder = 0
	ifEq := func(inner_b OrderType, inner_f int) {
		if inner_b == b && inner_f == f {
			arrivedAtOrder = 1
		}
	}
	var duration = 0

	switch e.Behaviour {
	case behaviourIdle:
		e.Movement = Requests_chooseDirection(e)
		if e.Movement == moveStop {
			return duration
		}
	case behaviourMoving:
		duration += utils.TRAVEL_TIME / 2
		e.Floor += int(e.Movement)
	case behaviourDoorOpen:
		duration -= utils.DOOR_OPEN_TIME / 2
	}

	for {
		if Requests_shouldStop(e) == 1 {
			e = request_clearAtCurrentFloor(e, ifEq, b, f)
			if arrivedAtOrder == 1 {
				return duration
			}
			duration += utils.DOOR_OPEN_TIME
			e.Movement = Requests_chooseDirection(e)
		}
		e.Floor += int(e.Movement)
		duration += utils.TRAVEL_TIME
	}
}

func Requests_shouldStop(e ElevatorState) int {
	switch e.Movement {
	case moveDown:
		shouldStop := (e.ActiveOrders[e.Floor][orderHallDown] == 1) ||
			(e.ActiveOrders[e.Floor][orderCab] == 1) ||
			(requests_below(e) != 1)
		if shouldStop {
			return 1
		} else {
			return 0
		}
	case moveUp:
		shouldStop := (e.ActiveOrders[e.Floor][orderHallUp] == 1 ||
			e.ActiveOrders[e.Floor][orderCab] == 1 ||
			requests_above(e) != 1)
		if shouldStop {
			return 1
		} else {
			return 0
		}

	default:
		return 1
	}
}

func Requests_chooseDirection(e ElevatorState) Movement {
	switch e.Movement {
	case moveUp:
		if requests_above(e) == 1 {
			return moveUp
		} else if requests_below(e) == 1 {
			return moveDown
		} else {
			return moveStop
		}
	case moveStop, moveDown:
		if requests_below(e) == 1 {
			return moveDown
		} else if requests_above(e) == 1 {
			return moveUp
		} else {
			return moveStop
		}
	default:
		return moveStop
	}
}

func request_clearAtCurrentFloor(e_old ElevatorState, onClearedOrder func(OrderType, int), b OrderType, floor int) ElevatorState {
	e := e_old
	var btn OrderType
	for btn = 0; btn < utils.ORDER_TYPE_NUM; btn++ {
		if e.ActiveOrders[e.Floor][btn] == 1 {
			e.ActiveOrders[e.Floor][btn] = 0
			if onClearedOrder != nil {
				onClearedOrder(btn, floor) //btn?
			}
		}
	}
	return e
}

func requests_above(e ElevatorState) int {
	for f := e.Floor + 1; f < utils.FLOOR_NUM; f++ {
		for btn := 0; btn < utils.ORDER_TYPE_NUM; btn++ {
			if e.ActiveOrders[f][btn] == 1 {
				return 1
			}
		}
	}
	return 0
}
func requests_below(e ElevatorState) int {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < utils.ORDER_TYPE_NUM; btn++ {
			if e.ActiveOrders[f][btn] == 1 {
				return 1
			}
		}
	}
	return 0
}
