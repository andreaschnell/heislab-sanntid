package eventManager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
)

type logSettings map[string]bool

type JsonEvent struct { //Finn ut navn
	TypeId string
	JSON   []byte
}

var JsonPublisherChannel chan interface{}
var addPublisherChannnel chan interface{}
var addSubscriberChannel chan interface{}

type subscribers map[reflect.Type][]interface{}

var LogSettings logSettings

// AddSubscribers add a publisher of an event. 
func AddPublishers(chans ...interface{}) {
	for _, ch := range chans {
		addPublisherChannnel <- ch
	}
}

// AddSubscribers add a subscribing channel to an event
func AddSubscribers(chans ...interface{}) {
	for _, ch := range chans {
		addSubscriberChannel <- ch
	}
}

// PublishJSON publishes json encoded event. 
func PublishJSON(JSON []byte, TypeID string) {
	d := JsonEvent{TypeID, JSON}
	JsonPublisherChannel <- d

}

// InitEventManager start the event manager. This has to be called before any publishers or subscribers are added.
func InitEventManager() {
	addPublisherChannnel = make(chan interface{})
	addSubscriberChannel = make(chan interface{})
	JsonPublisherChannel = make(chan interface{})
	go broker()
}

// broker takes incoming events through publisher channels and starts a routine to distribute to all subscriber channels. 
func broker() {

	subscribers := make(subscribers)
	LogSettings = loadLogSettings()

	selectCases := make([]reflect.SelectCase, 3)

	selectCases[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(addPublisherChannnel),
	}

	selectCases[1] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(addSubscriberChannel),
	}

	selectCases[2] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(JsonPublisherChannel),
	}

	for {
		chosen, value, _ := reflect.Select(selectCases)
		switch chosen {
		case 0:
			// add publisher
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(value.Interface()),
			})
		case 1:
			// add subscriber
			typ := value.Elem().Type().Elem()
			subscribers[typ] = append(subscribers[typ], value.Interface())
		case 2:
			// is an published json encoded event, unmarshal and distribute
			TypeID := value.Elem().Field(0).String()
			JSON, err := value.Elem().Field(1).Interface().([]byte)
			if !err { 
				panic("value not a []byte")
			}

			for T := range subscribers {
				typeName := T.String()
				if TypeID == typeName {

					v := reflect.New(T)
					json.Unmarshal([]byte(JSON), v.Interface())
					subs := subscribers[T]
					go distribute(v.Elem(), &subs)
				}
			}

		default:
			// is an published event, distribute
			subs := subscribers[value.Type()]
			go distribute(value, &subs)
		}
	}
}

// distributes to all subscribers
func distribute(value reflect.Value, s *[]interface{}) {
	i := value.Interface()
	logEvent(reflect.Indirect(value), LogSettings)
	for _, v := range *s {
		reflect.Select([]reflect.SelectCase{{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(v),
			Send: reflect.ValueOf(i),
		}})

	}
}

// logs events
func logEvent(evt reflect.Value, settings logSettings) {
	if !settings["Logging"] {
		return
	}

	if evt.Kind() == reflect.Struct {
		evtName := evt.Type().Name()
		if !settings[evtName+"Logging"] {
			return
		}
		fmt.Print(evtName)
		for i := 1; i < 30-len(evtName); i++ {
			fmt.Print(" ")
		}
		fmt.Print("|")
		logStruct(evt.Interface())
		fmt.Print("\n")
	}
}

// helper function to log structs
func logStruct(i interface{}) {
	s := reflect.ValueOf(i)

	for n := 0; n < s.NumField(); n++ {
		field := reflect.ValueOf(i).Field(n)
		fieldType := reflect.TypeOf(i).Field(n)
		if field.Kind() == reflect.Struct {

			fmt.Print(" ", field.Type().Name(), ": |")
			logStruct(field.Interface())
		} else if field.Kind() == reflect.Map {
			for _, e := range field.MapKeys() {
				v := field.MapIndex(e)
				fmt.Print(e, v) // how to get the value?
			}
		} else {

			fmt.Print(" ", fieldType.Name, ": ", field, "  ")
		}
	}

}

// loads log settings from file
func loadLogSettings() logSettings {
	settings := make(logSettings)

	raw, err := ioutil.ReadFile("eventLogSettings.json")
	if err != nil {
		panic(err.Error())
	}
	json.Unmarshal(raw, &settings)

	return settings
}
