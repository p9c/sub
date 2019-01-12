package sub

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"log"
	"math/rand"
	"net"
	"time"

	l "github.com/parallelcointeam/sub/clog"
)

// Implementations of common parts for node and worker

// NewBase creates a new base listener
func NewBase(cfg BaseCfg) (b *Base) {
	l.Trc.ch <- "creating new Base"
	b = &Base{
		cfg:       cfg,
		packets:   make(chan Packet, baseChanBufs),
		incoming:  make(chan Bundle, baseChanBufs),
		returning: make(chan Bundle, baseChanBufs),
		trash:     make(chan Bundle),
		quit:      make(chan bool),
	}
	return
}

// Start attempts to open a listener and commences receiving packets and assembling them into messages
func (b *Base) Start() (err error) {
	var addr *net.UDPAddr
	addr, err = net.ResolveUDPAddr(uNet, b.cfg.Listener)
	l.Check(err, l.Crt, "sub.Base.Start ResolveUDPAddr", true)
	b.listener, err = net.ListenUDP(uNet, addr)
	check(err, "sub.Base.Start ListenUDP", true)
	// Start up reader to push packets into packet channel
	go b.readFromSocket()
	go b.processPackets()
	go b.processBundles()
	go func() {
		for {
			select {
			case <-b.quit:
				break
			default:
			}
			select {
			case <-b.message:
				go b.cfg.Handler(<-b.message)
			}
		}
	}()
	return
}

// Stop shuts down the listener
func (b *Base) Stop() {
	b.quit <- true
	b.listener.Close()
}

func (b *Base) readFromSocket() {
	select {
	case <-b.quit:
		break
	}
	l[trc].ch <- "reading from sockets"
	for {
		var data = make([]byte, b.cfg.BufferSize)
		count, addr, err := b.listener.ReadFromUDP(data[0:])
		check(err, "sub.Base.readFromSocket.ReadFromUDP", false)
		data = data[:count]
		if count > 6 && err != nil {
			body := data[:count-4]
			check := data[count-4:]
			checkSum := binary.LittleEndian.Uint32(check)
			cs := crc32.Checksum(body, crc32.MakeTable(crc32.Castagnoli))
			if cs != checkSum {
				continue
			}
			l
			b.packets <- Packet{
				sender: addr,
				bytes:  data,
			}
		}
	}
}

func (b *Base) processPackets() {
	for {
		select {
		case <-b.quit:
			break
		default:
		}
		select {
		case <-b.packets:
			rand.Seed(time.Now().Unix())
			uuid := rand.Int31()
			p := <-b.packets
			sender := p.sender.String()
			go func() {
				for {
					select {
					case <-b.doneRet:
						for i := range b.returning {
							b.incoming <- i
						}
						break
					case <-b.returning:
						continue
					case <-b.trash:
						_ = <-b.trash
						continue
					}
				}
			}()
			for bi := range b.incoming {
				if bi.sender == sender {
					bi.packets = append(bi.packets, p.bytes)
					b.returning <- bi
					break
				}
				// If we have 3 or more it should be possible to now assemble the message
				if len(bi.packets) > 2 {
					b.incoming <- bi
					continue
				}
				// if
				if bi.received.Sub(time.Now()) < latencyMax {
					b.incoming <- bi
					continue
				} else {
					// delete all packets that fall outside the latency maximum
					b.trash <- bi
					break
				}
				b.doneRet <- true
			}
			continue
		}
	}
}

func (b *Base) processBundles() {
	for {
		select {
		case <-b.quit:
			break
		default:
		}
		var uuid int32
		select {
		case <-b.incoming:
			bundle := <-b.incoming
			data, err := rsDecode(bundle.packets)
			if err == nil &&
				bundle.uuid != uuid {
				rand.Seed(time.Now().Unix())
				uuid = rand.Int31()
				b.message <- Message{
					uuid:      bundle.uuid,
					sender:    bundle.sender,
					timestamp: bundle.received,
					bytes:     data,
				}
				uuid = bundle.uuid
				b.trash <- bundle
			}
		}
	}
}

// Send a message of up to maxMessageSize bytes to a given UDP address
func (b *Base) Send(data []byte, addr *net.UDPAddr) (err error) {
	if len(data) > 3072 {
		err = errors.New("maximum message size is " + fmt.Sprint(maxMessageSize) + " bytes")
	}
	addr, err = net.ResolveUDPAddr(uNet, addr.String())
	check(err, "sub.Base.Send.ResolveUDPAddr", false)
	conn, err := net.DialUDP(uNet, nil, addr)
	if check(err, "sub.Base.Send.DialUDP", false) {
		return
	}
	log.Print("'", string(data), "' -> ", addr)
	_, err = conn.Write(data)
	check(err, "sub.Base.Send.Write", false)
	return
}
