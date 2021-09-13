package main

import (
	"fmt"
	"runtime"
	"time"

	"./elevator"
	"./elevator/eventManager"
	"./elevator/log"
)

func main() {
	log.Init()
	fmt.Println("\n\n    ~('-'~) \\('-')/  Elevator Project Started \\('-')/ (~'-')~ \n\n\n")

	runtime.GOMAXPROCS(runtime.NumCPU())
	eventManager.InitEventManager()

	go elevator.ControllerModule()
	go elevator.AssignerModule()
	go elevator.ActiveOrdersModule()
	time.Sleep(1 * time.Second)
	go elevator.NetworkModule()
	time.Sleep(1 * time.Second)
	go elevator.DriverModule()

	elevator.StartElevator()
	fmt.Printf("\n\n")
	for {
		time.Sleep(time.Second)
	}
}
