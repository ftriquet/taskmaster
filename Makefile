clientname = client
servername = server

all: server client

server:
	go build -o $(servername) taskmaster/src_server


client:
	go build -o $(clientname) taskmaster/src_client

%.go: .tmp/%.gobj
	mkdir -p .tmp
	touch $<

clean:
	rm -rf $(servername) $(clientname)

.PHONY: client server
