// Package peer contains the networking code, which could be used ind
package peer

import (
	"encoding/json"
	"net"
	"sync"

	utils "github.com/Liamsi/Golang-P2PChat/utils"

	log "github.com/Sirupsen/logrus"
)

const port = ":1500"

// Output channel waiting on the user to type something
// TODO possibly rename this
var Output = make(chan string)

var (
	usersIPsMap         = make(map[string]string)   // list of users IPS connected to me
	usersConnectionsMap = make(map[string]net.Conn) // list of users connections connected to me
	myName              string                      // name of the client
	testing             = true
	mutex               = new(sync.Mutex)
)

// Message sent out to the server
type Message struct {
	Kind      string   //type of Message ("CONNECT","PRIVATE","PUBLIC","DISCONNECT","ADD")
	Username  string   //my username
	IP        string   //Ip address of my computer
	MSG       string   //Message
	Usernames []string //usernames of people connected
	IPs       []string //IP addresses of all the users connected
}

//sends Message to all peers
func (msg *Message) Send() {
	if testing {
		log.Println("send")
	}
	if testing {
		log.Println(usersConnectionsMap)
	}
	mutex.Lock()
	for _, peerConnection := range usersConnectionsMap {
		enc := json.NewEncoder(peerConnection)
		enc.Encode(msg)
	}
	mutex.Unlock()
}

// sends Message to a peer
func (msg *Message) SendPrivToUser(receiver string) {
	log.Debug("sendPrivToUser")
	if _, userExists := usersIPsMap[receiver]; userExists {
		peerConnection := usersConnectionsMap[receiver]
		enc := json.NewEncoder(peerConnection)
		enc.Encode(msg)
	} else {
		updateTextChan <- receiver + " is not a real user"
		//ctrl.updateText(receiver + " is not a real user")
	}
}

// channels
var (
	updateTextChan     chan string // init in Server(...)
	updateUserListChan chan []string
)

// RunServer is the part of the peer that acts like a server
// waits for possible peers to connect
func RunServer(updateTextCh chan string, updateUserList chan []string) {
	log.Debug("starting 'server'")
	updateTextChan = updateTextCh
	updateUserListChan = updateUserList

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

// receives Messages from peer
func receive(conn net.Conn) {

	defer conn.Close()
	dec := json.NewDecoder(conn)
	msg := new(Message)
	for {
		if err := dec.Decode(msg); err != nil {
			return
		}
		switch msg.Kind {
		case "CONNECT":
			log.Debug("Kind = CONNECT")
			if !handleConnect(*msg, conn) {
				return
			}
		case "PRIVATE":
			log.Debug("Kind = PRIVATE")
			updateTextChan <- "(private) from " + msg.Username + ": " + msg.MSG
			//ctrl.updateText("(private) from " + msg.Username + ": " + msg.MSG)
		case "PUBLIC":
			log.Debug("Kind = PUBLIC")
			updateTextChan <- msg.Username + ": " + msg.MSG
		case "DISCONNECT":
			log.Println("Kind = DISCONNECT")
			disconnect(*msg)
			return
		case "HEARTBEAT": //ask about it in the morning
			log.Println("HEARTBEAT")
		case "LIST":
			log.Debug("Kind = LIST")
			connectToPeers(*msg)
			return
		case "ADD":
			log.Debug("Kind = ADD")
			addPeer(*msg)
		}
	}
}

// handle a connection with a new peer
func handleConnect(msg Message, conn net.Conn) bool {

	log.Println("handleConnect")

	Users, IPs := utils.GetFromMap(usersIPsMap)
	Users = append(Users, myName)      //add my name to the list
	IPs = append(IPs, utils.GetMyIP()) //add my ip to the list
	response := Message{"LIST", "", "", "", Users, IPs}
	if _, usernameTaken := usersIPsMap[msg.Username]; usernameTaken {
		response.MSG = "Username already taken, choose another one that is not in the list"
		response.Send()
		return false
	}

	mutex.Lock()
	usersIPsMap[msg.Username] = msg.IP
	usersConnectionsMap[msg.Username] = conn
	mutex.Unlock()

	log.Println(usersConnectionsMap)
	response.SendPrivToUser(msg.Username)
	return true
}

// adds a peer to everyone list
func addPeer(msg Message) {

	mutex.Lock()
	usersIPsMap[msg.Username] = msg.IP
	conn := createConnection(msg.IP)
	usersConnectionsMap[msg.Username] = conn
	mutex.Unlock()

	userNames, _ := utils.GetFromMap(usersIPsMap)
	// TODO fix UI stuff
	updateUserListChan <- userNames
	updateTextChan <- msg.Username + " just joined the chat (from IP: " + msg.IP + ")"
	//	ctrl.updateList(userNames)
	//ctrl.updateText(msg.Username + " just joined the chat (from IP: " + msg.IP + ")")
}

//disconnect user by deleting him/her from list
func disconnect(msg Message) {
	mutex.Lock()
	delete(usersIPsMap, msg.Username)
	delete(usersConnectionsMap, msg.Username)
	mutex.Unlock()
	newUserList, _ := utils.GetFromMap(usersIPsMap)

	// TODO fix ui control stuff
	updateUserListChan <- newUserList
	updateTextChan <- msg.Username + " left the chat"
	// ctrl.updateList(newUserList)
	// ctrl.updateText(msg.Username + " left the chat")
}

// connects with everyone in the chat.
// The Message passed in contains a list of users and ips
func connectToPeers(msg Message) {
	for index, ip := range msg.IPs {
		conn := createConnection(ip)

		mutex.Lock()
		usersIPsMap[msg.Usernames[index]] = ip
		usersConnectionsMap[msg.Usernames[index]] = conn
		mutex.Unlock()
	}
	users, _ := utils.GetFromMap(usersIPsMap)
	// TODO UI Stuff
	updateUserListChan <- users
	//ctrl.updateList(users)
	addMessage := Message{"ADD", myName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0)}
	addMessage.Send()
}

// CreateConnection creates a new connection, given the IP address, and returns it
func createConnection(IP string) (conn net.Conn) {
	service := IP + port
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	utils.HandleErr(err)
	conn, err = net.DialTCP("tcp", nil, tcpAddr)
	utils.HandleErr(err)
	return
}

// IntroduceMyself introduces peer to the chat
func IntroduceMyself(IP string) {
	log.Println("introduceMyself")

	conn := createConnection(IP)
	enc := json.NewEncoder(conn)
	intromessage := Message{"CONNECT", myName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0)}
	err := enc.Encode(intromessage)
	if err != nil {
		log.Printf("Could not encode msg : %s", err)
	}
	go receive(conn)
}
