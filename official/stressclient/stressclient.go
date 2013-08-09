package main

import (
//	"P3-f12/official/tribclient"
//	"P3-f12/official/tribproto"
//	"flag"
//	"fmt"
//	"log"
//	"math/rand"
//	"os"
//	"strconv"
//	"strings"
//	"time"
)

func main() {
}
/*
var portnum *int = flag.Int("port", 9010, "server port # to connect to")
var clientId *string = flag.String("clientId", "0", "client id for user")
var numCmds *int = flag.Int("numCmds", 1000, "number of random commands to execute")
var seed *int64 = flag.Int64("seed", 0, "seed for random number generator used to execute commands")

func main() {

	flag.Parse()
	if flag.NArg() < 2 {
		log.Fatal("FAIL Usage: ./stressclient <user> <numTargets>")
	}

	serverAddress := "localhost"
	serverPort := fmt.Sprintf("%d", *portnum)
	client, _ := tribclient.NewTribbleclient(serverAddress, serverPort)

	user := flag.Arg(0)
	userNum, err := strconv.Atoi(user)
	if err != nil {
		log.Fatal("FAIL User %s not an integer", user)
	}
	numTargets, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		log.Fatal("FAIL numTargets invalid %s", flag.Arg(1))
	}

	client.CreateUser(user)
	if err != nil {
		log.Fatal("FAIL Error when making user %s", user)
		return
	}

	failed := false
	tribIndex := 0
	if *seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(*seed)
	}
	for i := 0; i < *numCmds; i++ {
		funcnum := rand.Intn(6)
		log.Printf("i:%d funcnum%d: ", i, funcnum)
		
		switch funcnum {
		case 0: //client.GetSubscription
			subscriptions, status, err := client.GetSubscriptions(user)
			if err != nil || status == tribproto.ENOSUCHUSER {
				failTest("error with GetSubscriptions")
			}
			failed = !validateSubscriptions(&subscriptions)
			//log.Printf("User%s Client%s: GetSubscriptions(User%s)", user, *clientId, user)
		case 1: //client.AddSubscription
			target := rand.Intn(numTargets)
			status, err := client.AddSubscription(user, strconv.Itoa(target))
			if err != nil || status == tribproto.ENOSUCHUSER {
				failTest("error with AddSubscription")
			}
			//log.Printf("User%s Client%s: AddSubscription(User%s, Target%d)", user, *clientId, user, target)
		case 2: //client.RemoveSubscription
			target := rand.Intn(numTargets)
			status, err := client.RemoveSubscription(user, strconv.Itoa(target))
			if err != nil || status == tribproto.ENOSUCHUSER {
				failTest("error with RemoveSubscription")
			}
			//log.Printf("User%s Client%s: RemoveSubscription(User%s, Target%d)", user, *clientId, user, target)
		case 3: //client.GetTribbles
			target := rand.Intn(numTargets)
			tribbles, _, err := client.GetTribbles(strconv.Itoa(target))
			if err != nil {
				failTest("error with GetTribbles")
			}
			failed = !validateTribbles(&tribbles, numTargets)
			//log.Printf("User%s Client%s: GetTribbles(Target%d)", user, *clientId, target)
		case 4: //client.PostTribble
			tribVal := userNum + tribIndex*numTargets
			msg := fmt.Sprintf("%d;%s", tribVal, *clientId)
			status, err := client.PostTribble(user, msg)
			if err != nil || status == tribproto.ENOSUCHUSER {
				failTest("error with PostTribble")
			}
			tribIndex++
			//log.Printf("User%s Client%s: PostTribble(User%s, $s)", user, *clientId, user, msg)
		case 5: //client.GetTribblesBySubscription
			tribbles, status, err := client.GetTribblesBySubscription(user)
			if err != nil || status == tribproto.ENOSUCHUSER {
				failTest("error with GetTribblesBySubscription")
			}
			failed = !validateTribbles(&tribbles, numTargets)
			//log.Printf("User%s Client%s: GetTribblesBySubscription(User%s)", user, *clientId, user)
		}
		if failed {
			failTest("tribbler output invalid")
		}
	}

	log.Print("PASS")
	os.Exit(7)
}

func failTest(msg string) {
	log.Fatal("FAIL ", msg)
}

// Check if there are any duplicates in the returned subscriptions
func validateSubscriptions(subscriptions *[]string) bool {
	subscriptionSet := make(map[string]bool, len(*subscriptions))

	for _, subscription := range *subscriptions {
		if subscriptionSet[subscription] == true {
			return false
		}
		subscriptionSet[subscription] = true
	}
	return true
}

func validateTribbles(tribbles *[]tribproto.Tribble, numTargets int) bool {
	userIdToLastVal := make(map[string]int, len(*tribbles))
	for _, tribble := range *tribbles {
		valAndId := strings.Split(tribble.Contents, ";")
		val, err := strconv.Atoi(valAndId[0])
		if err != nil {
			return false
		}
		user, err := strconv.Atoi(tribble.Userid)
		if err != nil {
			return false
		}
		userClientId := fmt.Sprintf("%s;%s", tribble.Userid, valAndId[1])

		lastVal := userIdToLastVal[userClientId]
		if val%numTargets == user && (lastVal == 0 || lastVal == val+numTargets) {
			userIdToLastVal[userClientId] = val
		} else {
			return false
		}
	}

	return true
}
*/
