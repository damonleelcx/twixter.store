# twixter.store

C:\Users\damon\kafka\bin\windows

1. Start Zookeeper

Kafka requires Zookeeper to run. Open Command Prompt, navigate to your Kafka installation directory, and execute:

cd path\to\kafka
zookeeper-server-start.bat C:\Users\damon\kafka\config\zookeeper.properties


2. Start Kafka Server

Open a new Command Prompt window, navigate again to your Kafka directory, and run:

cd path\to\kafka
kafka-server-start.bat C:\Users\damon\kafka\config\server.properties


go run main.go

Note:
1. Do not validate access with account type, validate with permissions only