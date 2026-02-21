BINARY = docviewer
PREFIX = $(HOME)/local/bin

.PHONY: build install clean

build:
	go build -o $(BINARY) .

install: build
	mkdir -p $(PREFIX)
	cp $(BINARY) $(PREFIX)/$(BINARY)

clean:
	rm -f $(BINARY)
