package testclient

import (
	"fmt"
	"regexp"
)

//TestMessage is a simple message implementation for use by the RestClient in composing messages for transmission to the server.
type TestMessage struct {
	Name string
	Text string
}

//Result returns a string of the message in the format used by the server.
func (t TestMessage) Result() string {
	return fmt.Sprintf("[%v]: %v", t.Name, t.Text)
}

//newTestMessage creates a new TestMessage.
func newTestMessage(name, message string) *TestMessage {
	msg := new(TestMessage)
	msg.Name = name
	msg.Text = message
	return msg
}

//RemoveTime removes the time from a message string from the server to make it easier to compare with the results.
func RemoveTime(s string) string {
	reg := regexp.MustCompile("\\A\\b\\d?\\d:\\d\\d[a,p]m\\b \\[")
	time := reg.FindString(s)
	if len(time) > 0 {
		return s[len(time)-1:]
	}
	return s
}
