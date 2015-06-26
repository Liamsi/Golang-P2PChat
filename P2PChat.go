package main

//worked with Matt Pozderac

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"os"
	"strings"
	"sync"

	"github.com/Liamsi/Golang-P2PChat/utils"
	"gopkg.in/qml.v1"
)

const port = ":1500"

var (
	output          = make(chan string)         //channel waitin on the user to type something
	listIPs         = make(map[string]string)   //list of users IPS connected to me
	listConnections = make(map[string]net.Conn) //list of users connections connected to me
	myName          string                      //name of the client
	testing         = true
	ctrl            control
	mutex           = new(sync.Mutex)
)

type control struct {
	Root        qml.Object
	convstring  string
	userlist    string
	inputString string
}

// message sent out to the server
type message struct {
	Kind      string   //type of message ("CONNECT","PRIVATE","PUBLIC","DISCONNECT","ADD")
	Username  string   //my username
	IP        string   //Ip address of my computer
	MSG       string   //message
	Usernames []string //usernames of people connected
	IPs       []string //IP addresses of all the users connected
}

//start the connection, introduces the user to the chat and creates graphical interface.
func main() {
	//adding myself to the list
	myName = os.Args[2]

	//starting graphics
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	engine := qml.NewEngine()
	component, err := engine.LoadFile("chat.qml")
	if err != nil {
		fmt.Println("no file to load for ui")
		fmt.Println(err.Error())
		os.Exit(0)
	}
	ctrl = control{convstring: ""}
	ctrl.convstring = ""
	context := engine.Context()
	context.SetVar("ctrl", &ctrl)

	win := component.CreateWindow(nil)

	win.Show() //show window
	ctrl.Root = win.Root()

	ctrl.updateText("Hello " + myName + ".\nFor private messages, type the message followed by * and the name of the receiver.\n To leave the conversation type disconnect")

	go server() //starting server
	if os.Args[1] != "127.0.0.1" {
		go introduceMyself(os.Args[1])
	} //connect to the first peer
	go userInput()

	win.Wait()
	closing := createmessage("DISCONNECT", myName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0))
	closing.send()
	return nil
}

//part of the peer that acts like a server

//waits for possible peers to connect
func server() {
	if testing {
		log.Println("server")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
	utils.ExitOnError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	utils.ExitOnError(err)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go receive(conn)
	}
}

//receives message from peer
func receive(conn net.Conn) {
	if testing {
		log.Println("receive")
	}
	defer conn.Close()
	dec := json.NewDecoder(conn)
	msg := new(message)
	for {
		if err := dec.Decode(msg); err != nil {
			return
		}
		switch msg.Kind {
		case "CONNECT":
			if testing {
				log.Println("Kind = CONNECT")
			}
			if !handleConnect(*msg, conn) {
				return
			}
		case "PRIVATE":
			if testing {
				log.Println("Kind = PRIVATE")
			}
			ctrl.updateText("(private) from " + msg.Username + ": " + msg.MSG)
		case "PUBLIC":
			if testing {
				log.Println("Kind = PUBLIC")
			}
			ctrl.updateText(msg.Username + ": " + msg.MSG)
		case "DISCONNECT":
			if testing {
				log.Println("Kind = DISCONNECT")
			}
			disconnect(*msg)
			return
		case "HEARTBEAT": //ask about it in the morning
			log.Println("HEARTBEAT")
		case "LIST":
			if testing {
				log.Println("Kind = LIST")
			}
			connectToPeers(*msg)
			return
		case "ADD":
			if testing {
				log.Println("Kind = ADD")
			}
			addPeer(*msg)
		}
	}
}

//introduces peer to the chat
func introduceMyself(IP string) {
	if testing {
		log.Println("introduceMyself")
	}
	conn := createConnection(IP)
	enc := json.NewEncoder(conn)
	intromessage := createmessage("CONNECT", myName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0))
	err := enc.Encode(intromessage)
	if err != nil {
		log.Printf("Could not encode msg : %s", err)
	}
	go receive(conn)
}

//handle a connection with a new peer
func handleConnect(msg message, conn net.Conn) bool {
	if testing {
		log.Println("handleConnect")
	}
	Users, IPs := getFromMap(listIPs)
	Users = append(Users, myName)      //add my name to the list
	IPs = append(IPs, utils.GetMyIP()) //add my ip to the list
	response := createmessage("LIST", "", "", "", Users, IPs)
	if alreadyAUser(msg.Username) {
		response.MSG = "Username already taken, choose another one that is not in the list"
		response.send()
		return false
	}
	mutex.Lock()
	listIPs[msg.Username] = msg.IP
	listConnections[msg.Username] = conn
	mutex.Unlock()
	log.Println(listConnections)
	response.sendPrivate(msg.Username)
	return true
}

//connects with everyone in the chat. The message passed in contains a list of users and ips
func connectToPeers(msg message) {
	for index, ip := range msg.IPs {
		conn := createConnection(ip)

		mutex.Lock()
		listIPs[msg.Usernames[index]] = ip
		listConnections[msg.Usernames[index]] = conn
		mutex.Unlock()
	}
	users, _ := getFromMap(listIPs)
	ctrl.updateList(users)
	addmessage := createmessage("ADD", myName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0))
	addmessage.send()
}

//adds a peer to everyone list
func addPeer(msg message) {

	mutex.Lock()
	listIPs[msg.Username] = msg.IP
	conn := createConnection(msg.IP)
	listConnections[msg.Username] = conn
	mutex.Unlock()

	userNames, _ := getFromMap(listIPs)
	ctrl.updateList(userNames)
	ctrl.updateText(msg.Username + " just joined the chat (from IP: " + msg.IP + ")")
}

//sends message to all peers
func (msg *message) send() {
	if testing {
		log.Println("send")
	}
	if testing {
		log.Println(listConnections)
	}
	mutex.Lock()
	for _, peerConnection := range listConnections {
		enc := json.NewEncoder(peerConnection)
		enc.Encode(msg)
	}
	mutex.Unlock()
}

//sends message to a peer
func (msg *message) sendPrivate(receiver string) {
	if testing {
		log.Println("sendPrivate")
	}
	if alreadyAUser(receiver) {
		peerConnection := listConnections[receiver]
		enc := json.NewEncoder(peerConnection)
		enc.Encode(msg)
	} else {
		ctrl.updateText(receiver + " is not a real user")
	}
}

//disconnect user by deleting him/her from list
func disconnect(msg message) {
	mutex.Lock()
	delete(listIPs, msg.Username)
	delete(listConnections, msg.Username)
	mutex.Unlock()
	newUserList, _ := getFromMap(listIPs)
	ctrl.updateList(newUserList)
	ctrl.updateText(msg.Username + " left the chat")
}

//returns two slices, the first one with the keys of the map and the second on with the values
func getFromMap(mappa map[string]string) ([]string, []string) {
	var keys []string
	var values []string
	for key, value := range mappa {
		keys = append(keys, key)
		values = append(values, value)
	}
	return keys, values
}

//creates a new connection, given the IP address, and returns it
func createConnection(IP string) (conn net.Conn) {
	service := IP + port
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	utils.HandleErr(err)
	conn, err = net.DialTCP("tcp", nil, tcpAddr)
	utils.HandleErr(err)
	return
}

//creates a new message using the parameters passed in and returns it
func createmessage(Kind string, Username string, IP string, MSG string, Usernames []string, IPs []string) (msg *message) {
	msg = new(message)
	msg.Kind = Kind
	msg.Username = Username
	msg.IP = IP
	msg.MSG = MSG
	msg.Usernames = Usernames
	msg.IPs = IPs
	return
}

//sends message to the server
func userInput() {
	if testing {
		log.Println("userInput")
	}
	msg := new(message)
	for {
		message := <-output
		log.Printf("userInput got message: %s", message)
		whatever := strings.Split(message, "*")
		if message == "disconnect" {
			msg = createmessage("DISCONNECT", myName, "", "", make([]string, 0), make([]string, 0))
			msg.send()
			break
		} else if len(whatever) > 1 {
			msg = createmessage("PRIVATE", myName, "", whatever[0], make([]string, 0), make([]string, 0))
			msg.sendPrivate(whatever[1])
			ctrl.updateText("(private) from " + myName + ": " + msg.MSG)
		} else {
			msg = createmessage("PUBLIC", myName, "", whatever[0], make([]string, 0), make([]string, 0))
			msg.send()
			ctrl.updateText(myName + ": " + msg.MSG)
		}
	}
	os.Exit(1)
}

//checks to see if a userName is already in the list
func alreadyAUser(user string) bool {
	for userName := range listIPs {
		if userName == user {
			return true
		}
	}
	return false
}

//Graphics methods

func (ctrl *control) TextEntered(text qml.Object) {
	//this method is called whenever a return key is typed in the text entry field.  The qml object calls this function
	ctrl.inputString = text.String("text") //the ctrl's inputString field holds the message
	//you will want to send it to the server
	//but for now just send it back to the conv field
	// ctrl.updateText(ctrl.inputString)
	output <- ctrl.inputString
}

func (ctrl *control) updateText(toAdd string) {
	//call this method whenever you want to add text to the qml object's conv field
	mutex.Lock()
	ctrl.convstring = ctrl.convstring + toAdd + "\n" //also keep track of everything in that field
	ctrl.Root.ObjectByName("conv").Set("text", ctrl.convstring)
	qml.Changed(ctrl, &ctrl.convstring)
	mutex.Unlock()
}

func (ctrl *control) updateList(list []string) {
	ctrl.userlist = ""
	for _, user := range list {
		ctrl.userlist += user + "\n"
	}
	ctrl.Root.ObjectByName("userlist").Set("text", ctrl.userlist)
	qml.Changed(ctrl, &ctrl.userlist)
}
