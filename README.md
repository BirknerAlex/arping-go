[![pkg.go.dev](https://godoc.org/github.com/BirknerAlex/arping-go?status.svg)](https://pkg.go.dev/github.com/BirknerAlex/arping-go)

# arping-go

Originally forked from https://github.com/j-keck/arping. This fork supports returning multiple mac addresses 
for a single ip address.
  
arping is a native go library to ping a host per arp datagram, or query a host mac address.

The currently supported platforms are: Linux and BSD.


## Usage
### arping library

* import this library per `import "github.com/BirknerAlex/arping-go"`
* export GOPATH if not already (`export GOPATH=$PWD`)
* download the library `go get`
* run it `sudo -E go run <YOUR PROGRAMM>` 
* or build it `go build`


The library requires raw socket access. So it must run as root, or with appropriate capabilities under linux: `sudo setcap cap_net_raw+ep <BIN>`.

For api doc and examples see: [godoc](http://godoc.org/github.com/BirknerAlex/arping-go) or check the standalone under 'cmd/arping/main.go'.


    
### arping executable
   
To get a runnable pinger use `go get -u github.com/BirknerAlex/arping-go/cmd/arping`. This will build the binary in $GOPATH/bin.

arping requires raw socket access. So it must run as root, or with appropriate capabilities under Linux: `sudo setcap cap_net_raw+ep <ARPING_PATH>`.

