package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	_ "github.com/pkg/profile"
	_ "go.einride.tech/can"
	"go.einride.tech/can/pkg/socketcan"
)

var lock = sync.RWMutex{}

func main() {
	//profile.Start(profile.ProfilePath("."))
	lastPing := time.Now()
	goodIDs := map[uint32]struct{}{}

	udpInPacket := make([]byte, 1024)
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
	//con.Read(udpInPacket)
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
			d, addr, _ := con.ReadFromUDP(udpInPacket)
			remoteAddr = addr
			//fmt.Print(d)
			if d == 4 {
				fmt.Printf("Received %x containing id %x number %d from %s\n", udpInPacket[0:d], udpInPacket[2:d], binary.BigEndian.Uint16(udpInPacket[2:d]), addr)
				lock.Lock()
				goodIDs[uint32(binary.BigEndian.Uint16(udpInPacket[2:d]))] = struct{}{}
				lock.Unlock()
			} else if d == 5 {
				con.WriteToUDP([]byte{00, 00, 192, 00, 240, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}, remoteAddr)
				lastPing = time.Now()
			}

		}
	}()

	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Printf("Usage %s [interface]...\n", os.Args[0])
		os.Exit(1)
	}

	for _, canInterface := range args {
		go func(ci string) {
			conn, _ := socketcan.DialContext(context.Background(), "can", ci)
			recv := socketcan.NewReceiver(conn)
			data := make([]byte, 16)

			//frame := &can.Frame{}
			for recv.Receive() {
				//*frame = recv.Frame()
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
		}(canInterface)
	}

	go func() {
		for {
			dnssd, err := zeroconf.Register("Notacanserver", "_panda._udp", "local.", 1338, []string{}, nil)
			if err != nil {
				panic(err)
			}

			for i := 0; i < 10; i++ {
				dnssd.SetText([]string{})
				time.Sleep(time.Second * 5)
			}
		}
	}()

	select {}
}
