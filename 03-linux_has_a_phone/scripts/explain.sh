#!/usr/bin/env bash
cat <<'EOF'
D-Bus mental model

    CLIENT
       |
       | destination = com.example.Lamp
       | path        = /com/example/Lamp
       | interface   = com.example.Lamp
       | member      = TurnOn
       v
    +----------------+
    |   D-Bus bus    |
    | route message  |
    +----------------+
       |
       v
    SERVER

A method call gets a reply.
A signal is an asynchronous event that subscribed clients can receive.
EOF
