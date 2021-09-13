Module overwiev
================


Modules
-------
- Assigner
- Contoller
- Driver
- Events
- Network
- Requests

Assigner
-----------------
This module recieves the cost result calculated for each elevator, and assigns the order to the elevator with the lowest cost. This assigned elevator is then again sent over the network to the other elevators, to compare if all the elevators agrees on which to send the order to. If all agrees, the assigned elevator takes the order, and if they disagree, all elevator takes the order to be sure it is handled.

Controller
-----------------
This module relates to an event-based "fsm". It knows the state of the elevator, and for each event it recieves, it decides what the elevator should do and send out the correct events for it to happend. 

Driver
-----------------
Most of this module is from given project resources for [driver-go](https://github.com/TTK4145/driver-go). It is however customized to send and recieve events to and from other modules. 

Network
-----------------
The Network module is based on the given project resources for [network-go](https://github.com/TTK4145/network-go). It is heavily modified. It broadcasts data over three ports. One port is for sending and receiving awake messages. If 100 consecutive awake messages one second apart from an elevator is lost, it is considered disconnected. A second channel is used to send and receive data packets. Last channel is used to send acknowledgements for data packets. If no ack for a sent packet is received it is resent a maximum of 30 times. Packet IDs are stored in order to prevent duplicates if ack messages are lost.

Requests
-----------------
This mudule contains a map to keep track on all the active hall orders for each elevator. In case of one elevator disconnecting, this module sends the hall orders assigned to the disconnected elevator to the remaining active elevators to be distributed. 

EventManager
-----------------
All modules communicate by using events, handled by the eventManager. The eventManager consist of publishers and subscribers. All data sent through a publisher channel will be sent to all regitered subscribers subscribing to the same channel type. The event Manager also has built in logging of all events. This can be turned on and off per even type in eventLogSettings.json 

Logging
-----------------
In addition to event logging, a loggin module is being used. This module prints in the following format:

[filename]       LEVEL |[Printed message] 

where filename is the file from where the print was executed and level is one of thre levels: Debug, Error and Info. 
Debug level is set in moduleLogSettings.json. When level is set to DBG all prints are printed. If level is set to 
ERR only Info and Error level prints will be printed. If level set to INF only Info level prints will be printed. 