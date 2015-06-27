package main

//worked with Matt Pozderac

import (
	"fmt"

	"os"

	"github.com/Liamsi/Golang-P2PChat/peer"
	"github.com/Liamsi/Golang-P2PChat/uicontrol"
	"github.com/Liamsi/Golang-P2PChat/utils"
	"gopkg.in/qml.v1"
)

var ctrl = control.Control{
	UpdatedTextToUI:   make(chan string, 10),
	UpdatedTextFromUI: make(chan string, 10),
	UpdateUserList:    make(chan []string, 10)}

// start the connection, introduces the user to the chat and creates graphical interface.
func main() {
	// adding myself to the list
	peer.MyName = os.Args[2]

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

	ctrl.UpdatedTextToUI <- "Hello " + peer.MyName + ".\nFor private messages, type the message followed by * and the name of the receiver.\n To leave the conversation type disconnect"

	go peer.RunServer(ctrl.UpdatedTextToUI, ctrl.UpdateUserList)

	if os.Args[1] != utils.GetMyIP() {
		go peer.IntroduceMyself(os.Args[1])
	} //connect to the first peer

	win.Wait()

	closing := peer.Message{"DISCONNECT", peer.MyName, utils.GetMyIP(), "", make([]string, 0), make([]string, 0)}
	closing.Send()
	return nil
}
