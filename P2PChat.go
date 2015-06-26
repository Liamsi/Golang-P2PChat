package main

//worked with Matt Pozderac

import (
	"fmt"
	"log"

	"os"
	"strings"

	"github.com/Liamsi/Golang-P2PChat/peer"
	"github.com/Liamsi/Golang-P2PChat/uicontrol"
	"github.com/Liamsi/Golang-P2PChat/utils"
	"gopkg.in/qml.v1"
)

var ctrl = control.Control{
	UpdateTextCh:   make(chan string),
	UpdateUserList: make(chan []string)}

var myName string

// start the connection, introduces the user to the chat and creates graphical interface.
func main() {
	// adding myself to the list
	myName = os.Args[2]

	// TODO flags instead of os.Args[]

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

	context := engine.Context()
	context.SetVar("ctrl", &ctrl)

	win := component.CreateWindow(nil)

	win.Show() //show window
	ctrl.Root = win.Root()

	go ctrl.StartControlLoop()

	ctrl.UpdateTextCh <- "Hello " + myName + ".\nFor private messages, type the message followed by * and the name of the receiver.\n To leave the conversation type disconnect"

	go peer.RunServer(ctrl.UpdateTextCh, ctrl.UpdateUserList)

	if os.Args[1] != "127.0.0.1" {
		go peer.IntroduceMyself(os.Args[1])
	} //connect to the first peer

	go userInput()

	win.Wait()

	closing := peer.Message{"DISCONNECT", myName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0)}
	closing.Send()
	return nil
}

// sends message to the server
func userInput() {

	log.Println("userInput")

	for {
		message := <-peer.Output
		log.Printf("userInput got message: %s", message)
		whatever := strings.Split(message, "*")
		if message == "disconnect" {
			msg := peer.Message{"DISCONNECT", myName, "", "", make([]string, 0), make([]string, 0)}
			msg.Send()
			break
		} else if len(whatever) > 1 {
			msg := peer.Message{"PRIVATE", myName, "", whatever[0], make([]string, 0), make([]string, 0)}
			msg.SendPrivToUser(whatever[1])
			ctrl.UpdateTextCh <- "(private) from " + myName + ": " + msg.MSG
		} else {
			msg := peer.Message{"PUBLIC", myName, "", whatever[0], make([]string, 0), make([]string, 0)}
			msg.Send()
			ctrl.UpdateTextCh <- myName + ": " + msg.MSG
		}
	}
	os.Exit(1)
}
