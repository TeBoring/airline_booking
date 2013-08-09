package tribimpl

import (
	"P3-f12/official/tribproto"
	"P3-f12/official/lsplog"
	"P3-f12/contrib/libconn"
	"P3-f12/contrib/libstore"
	"P3-f12/contrib/libairline"
	"P3-f12/contrib/airlineproto"
	"log"
	"fmt"
)

/* Implementation for Agency Server.
 * Key value pairs stored by agency server in the agency storage cluster are:
 * <userid>:new new_user
 * <userid>:bookings <FlightIds[]>
 */
 
type Tribserver struct {
        lib_store *libstore.Libstore //Used for communication with agency storage server
        lib_airline *libairline.Libairline //User for communication with airline storage server
        agencyid string
        myhostport string
}

func NewTribserver(agencyid, master, ownstorage, myhostport string, prefer int) *Tribserver {
	lsplog.SetVerbose(3)

	//Connect to master
	lc, err := libconn.NewLibconn(agencyid, master, myhostport, prefer)
	if lsplog.CheckReport(1, err) {
		return nil
	}
	
	//Init libstore : Agency storage server
	ls, err := libstore.NewLibstore(agencyid, ownstorage, myhostport, libstore.NONE)
	if lsplog.CheckReport(1, err) {
		log.Printf("[%s:%s] Fail to start", agencyid, myhostport)
		return nil
	}
	
	//Init libairline : Airline storage server
	la, err := libairline.NewLibairline(lc, agencyid, myhostport)
	if lsplog.CheckReport(1, err) {
		log.Printf("[%s:%s] Fail to start", agencyid, myhostport)
		return nil
	}
	
	//lsplog.Vlogf(5, "Server active... %s", myhostport);
	log.Printf("[%s:%s] Start", agencyid, myhostport)
	return &Tribserver{ls, la, agencyid, myhostport}
}

func (ts *Tribserver) CreateUser(args *tribproto.CreateUserArgs, reply *tribproto.CreateUserReply) error {
	//Check for existing user
	if ts.doesUserExist(args.Userid) {
		reply.Status = tribproto.EEXISTS
		return nil
	}
	
	// Create empty bookings entry for user
	cerr := ts.lib_store.Put(getNewUserKey(args.Userid), "new_user", ts.agencyid)
	perr := ts.lib_store.Put(getUserBookingsKey(args.Userid), "", ts.agencyid)
	if lsplog.CheckReport(3, cerr) || lsplog.CheckReport(3, perr) {
		return lsplog.MakeErr("Error creating user")
	}
	
	lsplog.Vlogf(5, "Created user:" + args.Userid);
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) ViewFlights(args *tribproto.ViewFlightsArgs, reply *tribproto.ViewFlightsReply) error {
	lsplog.Vlogf(5, "Get flights:" + args.From + " to:" + args.To + " date:"+args.DepartureDate);
	
	flightids, err := ts.lib_airline.GetAllFlights(args.From, args.To, args.DepartureDate)
	if lsplog.CheckReport(3, err) {
		return lsplog.MakeErr("Error fetching flight ids")
	}
	
	flightsInfo, ferr := ts.doGetFlightsFromIds(flightids)
	if lsplog.CheckReport(3, ferr) {
		return ferr
	}

	reply.Flights = flightsInfo
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) doGetFlightsFromIds(flightids []string) ([]airlineproto.FlightInfo, error) {
	flightsInfo := make([]airlineproto.FlightInfo, len(flightids))
	for i,fid := range flightids {
		f, err := ts.lib_airline.GetFlight(fid)
		lsplog.Vlogf(6, "Flight id " + fid)
		if lsplog.CheckReport(3, err) {
			return nil, lsplog.MakeErr("Error fetching flight from flight id")
		}
		flightsInfo[i] = f
	}
	
	return flightsInfo, nil
}

func (ts *Tribserver) AddBooking(args *tribproto.BookingArgs, reply *tribproto.BookingReply) error {
	lsplog.Vlogf(5, "Add booking: " + args.Userid + " flight :[0]" + args.FlightIds[0]);

	return ts.doBookingTransaction(false, args, reply)
}

func (ts *Tribserver) doBookingTransaction(isRemove bool, args *tribproto.BookingArgs, reply *tribproto.BookingReply) error {
	if !ts.doesUserExist(args.Userid) {
		reply.Status = tribproto.ENOSUCHUSER
		return nil
	}
	
	_, ferr := ts.doGetFlightsFromIds(args.FlightIds)
	if lsplog.CheckReport(3, ferr) {
		reply.Status = tribproto.EBOOKINGFAILED
		return nil
	}
	
	// Do not attempt to decrement count if duplicate booking
	
	flights, err := ts.lib_store.GetList(getUserBookingsKey(args.Userid), ts.agencyid)
	if lsplog.CheckReport(3, err) {
		reply.Status = tribproto.EBOOKINGFAILED
		return nil
	}
	
	fmt.Println("tribimpl@", flights)
	exist := false
	for _, fid1 := range args.FlightIds {
		for _,fid2 := range flights {
			if fid1 == fid2 {
				exist = true
				
			}
		}
	}
	
	if !isRemove {
		if exist == true {
			reply.Status = tribproto.EEXISTS
			return nil
		}
	} else {
		if exist == false {
			reply.Status = tribproto.ENOTBOOKED
			return nil
		}
	}
	
	
	err = ts.lib_airline.MakeBooking(args.FlightIds, isRemove)
	if err != nil {
		reply.Status = tribproto.EBOOKINGFAILED
		return nil
	}
	
	
	//If successful add to user bookings list
	for _,fid := range args.FlightIds {
		var err error
		if !isRemove {
			err = ts.lib_store.AppendToList(getUserBookingsKey(args.Userid), fid, ts.agencyid)
		} else {
		
			err = ts.lib_store.RemoveFromList(getUserBookingsKey(args.Userid), fid, ts.agencyid)
		}
		if lsplog.CheckReport(3, err) {
			reply.Status = tribproto.EBOOKINGFAILED
			return err
		}
	}
	
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) RemoveBooking(args *tribproto.BookingArgs, reply *tribproto.BookingReply) error {
	lsplog.Vlogf(5, "Remove booking: " + args.Userid + " flight :[0]" + args.FlightIds[0]);
	return ts.doBookingTransaction(true, args, reply)
}

func (ts *Tribserver) GetBookings(args *tribproto.GetBookingsArgs, reply *tribproto.GetBookingsReply) error {
	lsplog.Vlogf(5, "Get bookings: " + args.Userid);
	
	flightids, err := ts.lib_store.GetList(getUserBookingsKey(args.Userid), ts.agencyid)
	if lsplog.CheckReport(3, err) {
		reply.Status = tribproto.ENOSUCHUSER
		return nil
	}
	
	flightsInfo, ferr := ts.doGetFlightsFromIds(flightids)
	if lsplog.CheckReport(3, ferr) {
		return ferr
	}
	reply.Flights = flightsInfo
	reply.Status = tribproto.OK
	return nil
}

func (ts *Tribserver) AddFlight(args *tribproto.AddFlightArgs, reply *tribproto.AddFlightReply) error {
	
	err := ts.lib_airline.CreateFlight(args.Flight)
	if err == lsplog.MakeErr("Airline doesn't exist") {
		reply.Status = tribproto.ENOAIRLINE
		return nil
	}
	if err == lsplog.MakeErr("Flight already exist") {
		reply.Status = tribproto.EEXISTS
		return nil
	}
	
	
	
	reply.Status = tribproto.OK
	return nil
}

//Constant prefixes for different key types
const NEW_USER string = "n"
const ALL_BOOKINGS string = "b"
const SEP string = ":"

/* Generate different types of keys */
func getNewUserKey(userid string) string {
	return userid+SEP+NEW_USER
}

func getUserBookingsKey(userid string) string {
	return userid+SEP+ALL_BOOKINGS
}

/* Check if userid was created */
func  (ts *Tribserver) doesUserExist(userid string) bool {
	_, cerr := ts.lib_store.Get(getNewUserKey(userid), ts.agencyid)
	return cerr == nil
}
