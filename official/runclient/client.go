package main

import (
	"log"
	"fmt"
	"flag"
	"P3-f12/official/tribproto"
	"P3-f12/official/tribclient"
	"P3-f12/contrib/airlineproto"
	"strings"
	//"time"
)


// For parsing the command line
type cmd_info struct {
	cmdline string
	funcname string
	nargs int // number of required args
}
const (
	CMD_PUT = iota
	CMD_GET
)

var portnum *int = flag.Int("port", 9010, "server port # to connect to")

func main() {

	flag.Parse()
	if (flag.NArg() < 2) {
		log.Fatal("Insufficient arguments to client")
	}

	cmd := flag.Arg(0)
	
	serverAddress := "localhost"
	serverPort := fmt.Sprintf("%d", *portnum)
	client, _ := tribclient.NewTribbleclient(serverAddress, serverPort)

	cmdlist := []cmd_info {
		{ "uc", "CreateUser", 1 },
		{ "fv", "ViewFlights", 3 },
		{ "ba", "AddBooking", 2 },
		{ "br", "RemoveBooking", 2 },
		{ "bl", "GetBookings", 1 },
		{ "fa", "AddFlight", 7 },
		//{ "fd", "DeleteFlight", 1 },
	}

	cmdmap := make(map[string]cmd_info)
	for _, j := range(cmdlist) {
		cmdmap[j.cmdline] = j
	}

	ci, found := cmdmap[cmd]
	if (!found) {
		log.Fatal("Unknown command ", cmd)
	}
	if (flag.NArg() < (ci.nargs+1)) {
		log.Fatal("Insufficient arguments for ", cmd)
	}
	
	fmt.Printf("[Client] %s : %s\n", ci.funcname, strings.Join(flag.Args(), " "))

	switch(cmd) {
	case "uc":  // user create
		status, err := client.CreateUser(flag.Arg(1))
		PrintStatus(ci.funcname, status, err)
	case "fv":  // flight list
		flights, status, err := client.ViewFlights(flag.Arg(1), flag.Arg(2), flag.Arg(3))
		PrintStatus(ci.funcname, status, err)
		if (err == nil && status == tribproto.OK) {
			PrintFlights(flights)
		}
	case "ba":
		flights := make([]string, flag.NArg() - 2)
		for i:=2; i<flag.NArg(); i+=1 {
			flights[i-2] = flag.Arg(i)
		}
		status, err := client.AddBooking(flag.Arg(1), flights)
		PrintStatus(ci.funcname, status, err)
	case "br":  // booking remove
		flights := make([]string, flag.NArg() - 2)
		for i:=2; i<flag.NArg(); i+=1 {
			flights[i-2] = flag.Arg(i)
		}
		status, err := client.RemoveBooking(flag.Arg(1), flights)
		PrintStatus(ci.funcname, status, err)
	case "bl":  // booking list
		flights, status, err := client.GetBookings(flag.Arg(1))
		PrintStatus(ci.funcname, status, err)
		if (err == nil && status == tribproto.OK) {
			PrintFlights(flights)
		}
	case "fa":  
		status, err := client.AddFlight(flag.Arg(1), flag.Arg(2), flag.Arg(3), flag.Arg(4), flag.Arg(5), flag.Arg(6), flag.Arg(7))
		PrintStatus(ci.funcname, status, err)
	}
}

// This is a little lazy, but there are only 4 entries...
func TribStatusToString(status int) string {
	switch(status) {
	case tribproto.OK:
		return "OK"
	case tribproto.ENOSUCHUSER:
		return "No such user"
	case tribproto.ENOSUCHFLIGHT:
		return "No such flight"
	case tribproto.ENOSUCHAIRLINE:
		return "No such airline"
	case tribproto.EEXISTS:
		return "Already exists"
	case tribproto.EBOOKINGFAILED:
		return "Booking failed"
	case tribproto.ENOAIRLINE:
		return "Airline doesn't exist"
	case tribproto.ENOTBOOKED:
		return "Remove unbooked tickets"
		
	}
	return "Unknown error"
}

func PrintStatus(cmdname string, status int, err error) {
	if (status == tribproto.OK) {
		fmt.Printf("Operation:%s succeeded\n", cmdname)
	} else {
		fmt.Printf("Operation:%s failed: %s\n", cmdname, TribStatusToString(status))
	}
}

func PrintFlight(f airlineproto.FlightInfo) {
	fmt.Printf("%s - DATE:%s - FROM:%s - TO:%s - COST:%d - SEATS_LEFT:%d\n",
		f.ID, f.Departure, f.From, f.To, f.Cost, f.Slot)
}

func PrintFlights(flights []airlineproto.FlightInfo) {
	for _, f := range flights {
		PrintFlight(f)
	}
}
