package control

import (
	"fmt"
	"sync"

	"github.com/Liamsi/Golang-P2PChat/peer"

	"gopkg.in/qml.v1"
)

type Control struct {
	Root           qml.Object
	convString     string
	UpdateTextCh   chan string
	UpdateUserList chan []string
	userlist       string
	inputString    string
}

var (
	mutex = new(sync.Mutex)
)

func (ctrl *Control) StartControlLoop() {
	for {
		select {
		case update := <-ctrl.UpdateTextCh:
			fmt.Println("received text update", update)
			ctrl.updateText(update)
		case userListChanged := <-ctrl.UpdateUserList:
			fmt.Println("received userListChanged", userListChanged)
			ctrl.updateList(userListChanged)
		}
	}
}

func (ctrl *Control) TextEntered(text qml.Object) {
	//this method is called from qml whenever a return key is typed in the text entry field.
	ctrl.inputString = text.String("text") // the ctrl's inputString field holds the message
	// ctrl.updateText(ctrl.inputString)
	peer.Output <- ctrl.inputString
}

func (ctrl *Control) updateText(toAdd string) {
	//call this method whenever you want to add text to the qml object's conv field
	mutex.Lock()
	ctrl.convString = ctrl.convString + toAdd + "\n" //also keep track of everything in that field
	ctrl.Root.ObjectByName("conv").Set("text", ctrl.convString)
	qml.Changed(ctrl, &ctrl.convString)
	mutex.Unlock()
}

func (ctrl *Control) updateList(list []string) {
	ctrl.userlist = ""
	for _, user := range list {
		ctrl.userlist += user + "\n"
	}
	ctrl.Root.ObjectByName("userlist").Set("text", ctrl.userlist)
	qml.Changed(ctrl, &ctrl.userlist)
}
