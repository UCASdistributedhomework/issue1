package main

import (
	"bufio"
	"encoding/json"
    "fmt"
	"io"
    "io/ioutil"
    "log"
	"net"
	"os"
	"strings"
	"time"
)

type Inf struct {
	Name    string
	ThsPort string    `json:"thsPort"`
	Channel     []Seq `json:"channel"`
}

type Conn struct {
	state      int
	id         string
	localAddr  string
	nxtAddr []Seq
	note    *chan string				
	flag     bool
}

type Seq struct {
	NxtPort string `json:"nxtPort"`
	SeqName string `json:"seqName"`
}

func (pt *Conn) getState(){
	log.Print(pt.id, " ", "make snapshot", " ", " state= ",pt.state)
}

func NxtConn(id, localaddr string, nxtaddr []Seq, note *chan string) *Conn {
	return &Conn{
		state:      101,
		id:         id,
		localAddr:  localaddr,
		nxtAddr:    nxtaddr,
		note:       note,
		flag:       false,
	}
}

func getInf(filename string) (*Conn,error) {
	note := make(chan string)
	ori, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
		return nil,err
	}
	var inf Inf
	err = json.Unmarshal(ori, &inf)
	if err != nil {
		log.Fatal(err)
		return nil,err
	}
	conn := NxtConn(inf.Name, inf.ThsPort, inf.Channel, &note)
	fmt.Print(conn.id," state",conn.state," "," thisPort", conn.localAddr," ","nextProt",conn.nxtAddr[0].NxtPort,"\n")
	return conn,nil
}

func (pt *Conn) sendMsg(){
	for _, addr := range pt.nxtAddr {
		conn, err := net.Dial("tcp", addr.NxtPort)
		if err != nil {
			*pt.note <- "ERROR: " + err.Error()
			return
		}
		_, err = conn.Write([]byte(pt.id + "|msg is sending"))					
		if err != nil {
			log.Print(err)
			*pt.note <- "ERROR: " + err.Error()
			return
		}
	}
}

func (pt *Conn) SendMarker() {
	pt.flag = true
	pt.getState()
	for _, addr := range pt.nxtAddr {
		conn, err := net.Dial("tcp", addr.NxtPort)
		defer conn.Close()
		if err != nil {
			log.Print(err)
			*pt.note <- "ERROR: " + err.Error()
			return
		}
		_, err = conn.Write([]byte(pt.id + "|marker"))					
		if err != nil {
			log.Print(err)
			*pt.note <- "ERROR: " + err.Error()
			return
		}
	}
}

func (pt *Conn) rcvMsg(listener net.Listener) (bool,error){
	_, buffer, err := pt.rcv(listener)
	if err != nil {
		*pt.note <- "ERROR: " + err.Error()
		return false,err
	}
	split := strings.SplitN(string(buffer), "|", -1)
	if len(split) < 2 {
		return false,err
	}
	msg := split[1]
	if msg == "msg is sending" {
		if err != nil {
			*pt.note <- "ERROR: " + err.Error()
			return  false,err
		}
		pt.state +=1
		return true,nil
	}
	if msg == "marker" && !pt.flag {
		pt.SendMarker()
		return false,nil
	}
	return false,nil
}

func (pt *Conn) rcv(listener net.Listener) (string, []byte, error) {
	conn, err := listener.Accept()
	if err != nil {
		*pt.note <- "ERROR: " + err.Error()
		return "", []byte{}, err
	}
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil && err != io.EOF {
		*pt.note <- "ERROR: " + err.Error()
		return "", []byte{}, err
	}
	return conn.RemoteAddr().String(), buffer[:n], nil
}


func (pt *Conn) rotation(){
	listener, err := net.Listen("tcp", pt.localAddr)
	if err != nil {
		*pt.note <- "ERROR: " + err.Error()
		log.Println(err.Error())
		return
	}
	for {
		msg,err := pt.rcvMsg(listener); if err != nil{
			continue
		}
		if msg == true{
			time.Sleep(2 * time.Second)
			pt.sendMsg()
		}
	}
}

func main() {
	prcs1,err := getInf("procsP.json");
	if err != nil{
		log.Fatal(err)
		return
	}
	prcs2,err2 := getInf("procsQ.json");
	if err2 != nil{
		log.Fatal(err2)
		return
	}

	go func() {
		prcs1.rotation()
	}()

	go func() {
		prcs2.rotation()
	}()

	time.Sleep(2 * time.Second)
	prcs1.sendMsg()					
	fmt.Print("input S to start\n")
	for{
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		if text[:len(text)-1] == "s"{
			prcs1.SendMarker()
		}
		if text[:len(text)-1] == "c"{
			prcs1.flag = false
			prcs2.flag = false
		}
	}
	return
}
