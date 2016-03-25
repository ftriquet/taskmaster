clientname = client
servername = server

SRC_SERVER=$(shell ls src_server/{methods,server,html}.go)
SRC_CLIENT=$(shell ls src_client/{method,client}.go)

all: server client
server: $(SRC_SERVER)
	go build -o $(servername) -race taskmaster/src_server

client: $(SRC_CLIENT)
	go build -o $(clientname) -race taskmaster/src_client

clean:
	rm -rf $(servername) $(clientname)

.PHONY: client server
