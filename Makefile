clientname = client
servername = server

all: server client
server:
	go build -o $(servername) -race taskmaster/src_server

client:
	go build -o $(clientname) -race taskmaster/src_client

clean:
	rm -rf $(servername) $(clientname)

.PHONY: client server
