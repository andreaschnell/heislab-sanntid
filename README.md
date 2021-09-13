Elevator Project
================

Structure
-----------------
The structure of this project is event based. Each module can subscribe and publish to an event channel, and take action accordinly based on what event is happening. 
The main function serves as a start-up routine for all modules, and dependencies between the modules are shown by the subscribers and publishers initialized in the module functions.
The communication between the elevators is bradcasting over UDP, resulting in a mesh network, where each elevator has to be initialized with an unique elevator ID. 

Resources and libraries
-------
The cost function is heavily based on the one published in [the project resources repository](https://github.com/TTK4145/Project-resources) and [elev_algo/requests.c](https://github.com/TTK4145/Project-resources/blob/master/elev_algo/requests.c), but translated to Go. The [connection module](/elevator/conn) is copyed from [the network repository](https://github.com/TTK4145/Network-go/tree/master/network/conn), and the [network module](/elevator/network.go) is based on the [bcast](https://github.com/TTK4145/Network-go/tree/master/network/bcast), but with a significant amount of additional features and some modifications. 

No further resources were used except from standard Go libraries.

To run the program
---------------
Windows:
- We have created a Makefile with different make commans relating to how many elevators you want to run.
- If you want to run multiple elevators on one computer, use `make RunSimulatorX` and then `make BuildAndRunX`, where X is number of elevators.
- If you only want to run one elevator, but with a specific ID, use `make RunSimAloneX` and then `make BuildAloneX`. Where X is the wanted elevator ID.
- It is important to choose different IDs if you want to test the system on different computers over the network

Linux:
- To open the simulator: `gnome-terminal -- ./SimElevatorServer --port xxxxx`
- To run the program: `gnome-terminal -- go run main.go -id X -port xxxxx`
- Where X is the wanted Elevator ID, and xxxxx is the port you want to use. E.g. id = 0 and port = 12067. Change xxxxx if you want to run another elevator. E.g. if id = 1, use port = 12068.