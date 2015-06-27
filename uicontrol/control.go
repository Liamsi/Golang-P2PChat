package control

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Liamsi/Golang-P2PChat/peer"

	"gopkg.in/qml.v1"
)

// Control type holds the qml object and all necessary channels to communicate
// with the rest of the application
type Control struct {
	Root              qml.Object
	UpdatedTextToUI   chan string
	UpdatedTextFromUI chan string
	UpdateUserList    chan []string
	conversationStr   string
	userlist          string
	inputString       string
}

var (
	mutex = new(sync.Mutex)
)

// StartControlLoop checks the control channels for updates and updates the UI accordingly
func (ctrl *Control) StartControlLoop() {
	fmt.Println("Running Control loop")
	for {
		select {
		case update := <-ctrl.UpdatedTextToUI:
			fmt.Println("received text update", update)
			ctrl.updateText(update)
		case userListChanged := <-ctrl.UpdateUserList:
			fmt.Println("received userListChanged", userListChanged)
			ctrl.updateList(userListChanged)
		case updateFromUI := <-ctrl.UpdatedTextFromUI:
			fmt.Println("<-ctrl.UpdatedTextFromUI", updateFromUI)
			ctrl.handleUserInput(updateFromUI)
		default:
			// carry on
		}
	}
}

// sends message to the server
func (ctrl *Control) handleUserInput(input string) {

	log.Println("userInput")
	log.Printf("userInput got message: %s", input)
	whatever := strings.Split(input, "*")
	if input == "disconnect" {
		msg := peer.Message{"DISCONNECT", peer.MyName, "", "", make([]string, 0), make([]string, 0)}
		msg.Send()
		//os.Exit(1)
	} else if len(whatever) > 1 {
		msg := peer.Message{"PRIVATE", peer.MyName, "", whatever[0], make([]string, 0), make([]string, 0)}
		msg.SendPrivToUser(whatever[1], ctrl.UpdatedTextFromUI)
		ctrl.UpdatedTextToUI <- "(private) from " + peer.MyName + ": " + msg.MSG
	} else {
		msg := peer.Message{"PUBLIC", peer.MyName, "", whatever[0], make([]string, 0), make([]string, 0)}
		msg.Send()
		ctrl.UpdatedTextToUI <- peer.MyName + ": " + msg.MSG
	}

}

// TextEntered is called from qml whenever a return key is typed in the text entry field.
func (ctrl *Control) TextEntered(text qml.Object) {
	textStr := text.String("text")
	fmt.Println("User entered text", textStr)
	ctrl.UpdatedTextFromUI <- textStr
}

func (ctrl *Control) updateText(toAdd string) {
	//call this method whenever you want to add text to the qml object's conv field
	mutex.Lock()
	ctrl.conversationStr = ctrl.conversationStr + toAdd + "\n" //also keep track of everything in that field
	ctrl.Root.ObjectByName("conv").Set("text", ctrl.conversationStr)
	qml.Changed(ctrl, &ctrl.conversationStr)
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
