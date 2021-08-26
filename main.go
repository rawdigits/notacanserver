package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"go.einride.tech/can/pkg/socketcan"
)

var lock = sync.RWMutex{}

func main() {
	lastPing := time.Now()
	goodIDs := map[uint32]struct{}{}
	// Error handling omitted to keep example simple
	blah := make([]byte, 1024)
	addr := net.UDPAddr{
		Port: 1338,
		IP:   net.ParseIP(""),
	}
	remoteAddr := &net.UDPAddr{}
	con, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer con.Close()
	//con.Read(blah)
	go func() {
		for {
			if time.Now().After(lastPing.Add(5 * time.Second)) {
				remoteAddr = &net.UDPAddr{}
				lock.Lock()
				goodIDs = map[uint32]struct{}{}
				lock.Unlock()
			}
			time.Sleep(time.Second * 1)
		}
	}()

	go func() {
		for {
			d, addr, _ := con.ReadFromUDP(blah)
			remoteAddr = addr
			//fmt.Print(d)
			if d == 4 {
				fmt.Printf("Received %x containing id %x number %d from %s\n", blah[0:d], blah[2:d], binary.BigEndian.Uint16(blah[2:d]), addr)
				lock.Lock()
				goodIDs[uint32(binary.BigEndian.Uint16(blah[2:d]))] = struct{}{}
				lock.Unlock()
			} else if d == 5 {
				con.WriteToUDP([]byte{00, 00, 192, 00, 240, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}, remoteAddr)
				lastPing = time.Now()
			}

		}
	}()

	args := os.Args[1:]

	conn, _ := socketcan.DialContext(context.Background(), "can", args[0])
	recv := socketcan.NewReceiver(conn)
	data := make([]byte, 16)

	go func() {
		for recv.Receive() {
			frame := recv.Frame()
			lock.RLock()
			if _, ok := goodIDs[frame.ID]; !ok {
				lock.RUnlock()
				continue
			}
			lock.RUnlock()
			binary.LittleEndian.PutUint32(data, uint32(frame.ID)<<21)
			binary.LittleEndian.PutUint32(data[4:], uint32(frame.Length))
			binary.LittleEndian.PutUint64(data[8:], frame.Data.PackLittleEndian())
			con.WriteToUDP(data, remoteAddr)

		}
	}()

	// TODO: i hacked in can1 and this needs to be better
	conn2, _ := socketcan.DialContext(context.Background(), "can", "can1")
	recv2 := socketcan.NewReceiver(conn2)

	go func() {
		for recv2.Receive() {
			frame := recv2.Frame()
			lock.RLock()
			if _, ok := goodIDs[frame.ID]; !ok {
				lock.RUnlock()
				continue
			}
			lock.RUnlock()
			binary.LittleEndian.PutUint32(data, uint32(frame.ID)<<21)
			binary.LittleEndian.PutUint32(data[4:], uint32(frame.Length))
			binary.LittleEndian.PutUint64(data[8:], frame.Data.PackLittleEndian())
			con.WriteToUDP(data, remoteAddr)

		}
	}()

	select {}
}
