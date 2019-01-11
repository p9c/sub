package sub

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
)

// Implementations of common parts for node and worker

// NewBase creates a new base listener
func NewBase(cfg BaseCfg) (b *Base) {
	b = &Base{
		cfg:      cfg,
		packets:  make(chan *Packet),
		Messages: make(chan *Message),
		kill:     make(chan bool),
	}
	return
}

// Start attempts to open a listener and commences receiving packets and assembling them into messages
func (b *Base) Start() (err error) {
	var addr *net.UDPAddr
	addr, err = net.ResolveUDPAddr(uNet, b.cfg.Listener)
	check(err, "sub.Base.Start ResolveUDPAddr", true)
	b.listener, err = net.ListenUDP(uNet, addr)
	check(err, "sub.Base.Start ListenUDP", true)
	// Start up reader to push packets into packet channel
	go b.readFromSocket()
	return
}

// Stop shuts down the listener
func (b *Base) Stop() {
	b.kill <- true
	b.listener.Close()
}

func (b *Base) readFromSocket() {
	for {
		var data = make([]byte, b.cfg.BufferSize)
		count, addr, err := b.listener.ReadFromUDP(data[0:])
		check(err, "sub.Base.readFromSocket.ReadFromUDP", false)
		sender := b.listener.RemoteAddr().(*net.UDPAddr)
		log.Println("received packets from", sender)
		data = data[:count]
		// packet data is terminated with a 32 bit CRC32 checksum for quickly identifying potentially corrupted packets, and a 16 bit length prefix
		if count > 8 && err != nil {
			log.Print("'", string(data), "' <- ", addr)
			packet := &Packet{
				bytes:  data[:count-5],
				check:  binary.LittleEndian.Uint32(data[count-5:]),
				sender: sender,
			}
			b.packets <- packet
			select {
			case <-b.kill:
				break
			default:
			}
		}
	}
}

// Send a message of up to maxMessageSize bytes to a given UDP address
func (b *Base) Send(data []byte, addr *net.UDPAddr) (err error) {
	if len(data) > 3072 {
		err = errors.New("maximum message size is " + fmt.Sprint(maxMessageSize) + " bytes")
	}
	return
}
