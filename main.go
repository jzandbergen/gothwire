package main

import (
	"embed"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

//go:embed web/*
var fs embed.FS

// msgObject is a object that is created after receiving a websocket message.
// it creates a simple HTML template that can be rendered serverside as a
// fragment after which the rendered html is send back to the client.
//
// The client then finds the parent_target element and adds this rendered
// element.
//
// HTML over the Wire aka HOTWire.
type msgObject struct {
	Uuid     string
	Message  string
	template *template.Template
}

// Instantiate a new msgObject.
func NewMsgObject(msg string) (msgObject, error) {
	o := msgObject{
		Uuid:    uuid.New().String(),
		Message: msg,
	}

	templateContent := `
	<div id={{.Uuid}} parent_target="root">
		<b>i got:</b> {{ .Message }}
	</div><br>
	`

	tmpl, err := template.New("myTemplate").Parse(templateContent)
	if err != nil {
		return o, err
	}
	o.template = tmpl

	return o, nil
}

// Render the HTML fragment.
func (m *msgObject) Render() []byte {
	tmplBuffer := new(strings.Builder)
	if err := m.template.Execute(tmplBuffer, m); err != nil {
		log.Printf("dikke error: %v", err)
	}
	return ([]byte(tmplBuffer.String()))
}

// serveIndex serves the index page.
func serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("web/static/index.html")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, string(data))
}

// handleWebSocket handles each websocket connection.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	defer conn.Close()

	for {
		// Read message from client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Failed to read message:", err)
			break
		}

		log.Printf("Received message: %s\n", msg)

		m, err := NewMsgObject(string(msg))
		if err != nil {
			log.Fatal(err)
		}

		// Write message back to client
		err = conn.WriteMessage(websocket.TextMessage, m.Render())
		if err != nil {
			log.Println("Failed to write message:", err)
			break
		}
	}
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/", serveIndex)
	router.HandleFunc("/ws", handleWebSocket)
	log.Fatal(http.ListenAndServe(":8083", router))
}
