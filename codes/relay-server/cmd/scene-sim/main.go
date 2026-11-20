// Command scene-sim emulates a Decentraland scene connected to the relay.
// It prints every message received and sends commands either from the
// -send flag (scripted) or typed on stdin (interactive).
//
// Command format: "base shoulder elbow gripper" in degrees, separated by
// spaces or commas, e.g. "45 120 60 10". "home" sends 90 90 90 90.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"pibic-digital-twins/relay-server/internal/relay"
)

func main() {
	addr := flag.String("url", "ws://localhost:8080/ws/scene", "relay scene WebSocket URL")
	token := flag.String("token", os.Getenv("SCENE_TOKEN"), "scene token (defaults to $SCENE_TOKEN)")
	send := flag.String("send", "", `commands to send, separated by ";" (e.g. "45 120 60 10;home"); empty reads stdin`)
	interval := flag.Duration("interval", 3*time.Second, "delay between scripted commands")
	wait := flag.Duration("wait", 5*time.Second, "how long to keep listening after the last scripted command")
	flag.Parse()

	u, err := url.Parse(*addr)
	if err != nil {
		log.Fatalf("invalid -url: %v", err)
	}
	if *token != "" {
		q := u.Query()
		q.Set("token", *token)
		u.RawQuery = q.Encode()
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()
	log.Printf("connected to %s", *addr)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("connection closed: %v", err)
				}
				return
			}
			log.Printf("<- %s", msg)
		}
	}()

	var id int
	sendCmd := func(line string) {
		state, err := parseCommand(line)
		if err != nil {
			log.Printf("ignored %q: %v", line, err)
			return
		}
		id++
		env := relay.Envelope{Type: relay.TypeCommand, ID: strconv.Itoa(id), State: &state}
		raw, _ := json.Marshal(env)
		log.Printf("-> %s", raw)
		if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			log.Fatalf("write: %v", err)
		}
	}

	if *send != "" {
		for i, line := range strings.Split(*send, ";") {
			if i > 0 {
				time.Sleep(*interval)
			}
			sendCmd(line)
		}
		select {
		case <-time.After(*wait):
		case <-done:
		}
		closeGracefully(conn, done)
		return
	}

	fmt.Fprintln(os.Stderr, `type "base shoulder elbow gripper" (degrees) or "home"; Ctrl+D to quit`)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			sendCmd(line)
		}
	}
	closeGracefully(conn, done)
}

// closeGracefully sends a close frame and waits briefly for the relay to
// close its side, so the relay sees a normal closure.
func closeGracefully(conn *websocket.Conn, done <-chan struct{}) {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	_ = conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}

func parseCommand(line string) (relay.ArmState, error) {
	line = strings.TrimSpace(line)
	if line == "home" {
		line = "90 90 90 90"
	}
	fields := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == ',' })
	if len(fields) != relay.JointCount+1 {
		return relay.ArmState{}, fmt.Errorf("expected %d angles, got %d", relay.JointCount+1, len(fields))
	}
	vals := make([]float64, len(fields))
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return relay.ArmState{}, err
		}
		vals[i] = v
	}
	return relay.ArmState{Joints: vals[:relay.JointCount], Gripper: vals[relay.JointCount]}, nil
}
