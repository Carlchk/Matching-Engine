# Q2. Matching Engine

# How to run?

Just simply run ```docker-compose up``` to start all the services.

Noted that port ```3000```,```3001```, ```4001```, ```4002``` ```5672``` and ```15672``` are not being used by other application.

----

# Technical details 

## Tech Stack

Frontend web applciation: SolidJS

Order handling API: Golang

Matching Engine: Golang

Message Queue Services: RabbitMQ

## Architecture

![Architecture](arch.png)