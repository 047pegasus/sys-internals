# D-Bus Demo — Go + Linux

A small, readable demonstration of D-Bus inter-process communication.

## Mental model

Think of a building with many rooms and one reception desk. A room does not
need to know another room's phone number or exact location. It tells reception
who it wants to reach and reception routes the message.

D-Bus works similarly:

    Client process
          |
          | structured message
          v
       D-Bus bus
          |
          | route by well-known name
          v
    Server process

This demo creates a tiny `Lamp` service. The client turns it on, changes its
brightness, asks for status, and listens for a `StateChanged` signal.

## D-Bus pieces

    com.example.Lamp       service / well-known bus name
    /com/example/Lamp      object path
    com.example.Lamp       interface
    TurnOn                 method
    StateChanged           signal

## Requirements

Linux, Go 1.22+, and D-Bus.

Run:

    ./scripts/run.sh

Inspect a running bus with:

    busctl --user list
    busctl --user tree com.example.Lamp
    busctl --user introspect com.example.Lamp /com/example/Lamp

This is an educational project, intentionally kept small and readable.

###Reading Material:

https://www.freedesktop.org/wiki/Software/dbus/

https://www.baeldung.com/linux/dbus

https://unix.stackexchange.com/questions/604258/what-is-d-bus-practically-useful-for
