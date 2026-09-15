package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	serviceName   = "com.example.Lamp"
	objectPath    = dbus.ObjectPath("/com/example/Lamp")
	interfaceName = "com.example.Lamp"
)

type Lamp struct {
	on         bool
	brightness int32
	conn       *dbus.Conn
}

func (l *Lamp) TurnOn() *dbus.Error {
	l.on = true
	if l.brightness == 0 { l.brightness = 100 }
	log.Println("A method call arrived: TurnOn()")
	log.Println("The lamp is now ON")
	l.emitState()
	return nil
}

func (l *Lamp) TurnOff() *dbus.Error {
	l.on = false
	log.Println("A method call arrived: TurnOff()")
	log.Println("The lamp is now OFF")
	l.emitState()
	return nil
}

func (l *Lamp) SetBrightness(value int32) *dbus.Error {
	if value < 0 || value > 100 {
		return dbus.MakeFailedError(fmt.Errorf("brightness must be between 0 and 100"))
	}
	l.brightness = value
	log.Printf("A method call arrived: SetBrightness(%d)", value)
	log.Printf("Brightness is now %d%%", value)
	l.emitState()
	return nil
}

func (l *Lamp) Status() (bool, int32, *dbus.Error) {
	log.Println("A method call arrived: Status()")
	log.Printf("Sending reply: on=%t brightness=%d%%", l.on, l.brightness)
	return l.on, l.brightness, nil
}

func (l *Lamp) emitState() {
	if err := l.conn.Emit(objectPath, interfaceName+".StateChanged", l.on, l.brightness); err != nil {
		log.Printf("could not emit StateChanged: %v", err)
		return
	}
	log.Println("Broadcasting StateChanged so interested clients can react")
}

func runServer() {
	log.SetPrefix("[SERVER] ")
	log.SetFlags(0)

	conn, err := dbus.ConnectSessionBus()
	if err != nil { log.Fatal("could not connect to D-Bus: ", err) }
	defer conn.Close()

	reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil { log.Fatal("could not request service name: ", err) }
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("another process already owns ", serviceName)
	}

	lamp := &Lamp{brightness: 100, conn: conn}
	if err := conn.Export(lamp, objectPath, interfaceName); err != nil {
		log.Fatal("could not export object: ", err)
	}

	log.Println("Connected to the D-Bus session bus")
	log.Printf("I now own the service name %s", serviceName)
	log.Printf("I exported an object at %s", objectPath)
	log.Println("Waiting for another process to send messages...")
	select {}
}

func runClient() {
	log.SetPrefix("[CLIENT] ")
	log.SetFlags(0)

	time.Sleep(700 * time.Millisecond)

	conn, err := dbus.ConnectSessionBus()
	if err != nil { log.Fatal("could not connect to D-Bus: ", err) }
	defer conn.Close()

	log.Println("Connected to the same D-Bus session bus")
	log.Printf("Looking for %s...", serviceName)

	obj := conn.Object(serviceName, objectPath)
	log.Println("I can address the service by name — no PID or custom socket lookup needed")

	// Listen for the server's asynchronous signal.
	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(objectPath),
		dbus.WithMatchInterface(interfaceName),
		dbus.WithMatchMember("StateChanged"),
	); err != nil {
		log.Fatal("could not subscribe to StateChanged: ", err)
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	go func() {
		for sig := range signals {
			if len(sig.Body) == 2 {
				log.Printf("Received asynchronous StateChanged: lamp_on=%t brightness=%d%%",
					sig.Body[0], sig.Body[1])
			}
		}
	}()

	log.Println("Calling TurnOn()")
	if err := obj.Call(interfaceName+".TurnOn", 0).Err; err != nil {
		log.Fatal("TurnOn failed: ", err)
	}
	log.Println("The method call completed successfully")

	log.Println("Calling SetBrightness(65)")
	if err := obj.Call(interfaceName+".SetBrightness", 0, int32(65)).Err; err != nil {
		log.Fatal("SetBrightness failed: ", err)
	}

	var on bool
	var brightness int32
	log.Println("Calling Status() and waiting for a reply")
	if err := obj.Call(interfaceName+".Status", 0).Store(&on, &brightness); err != nil {
		log.Fatal("Status failed: ", err)
	}
	log.Printf("The reply says: lamp_on=%t brightness=%d%%", on, brightness)

	time.Sleep(250 * time.Millisecond)
	log.Println("Finished — two independent processes communicated through D-Bus")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: dbus-demo server|client")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "server": runServer()
	case "client": runClient()
	default:
		fmt.Println("usage: dbus-demo server|client")
		os.Exit(2)
	}
}
