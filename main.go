// vim: et:ts=4:sw=4
package main

import (
    "github.com/songgao/water"
    "golang.org/x/net/ipv4"
    "golang.org/x/net/ipv6"
    "log"
    "github.com/jdewald/ipoveremail/pkg/ipoe"
)


func main() {
    name := "ipoveremail0"
    config := water.Config { 
        DeviceType: water.TUN,
    }
    config.Name = name
    config.Persist = false
    config.MultiQueue = true

    intf, err := water.New(config)
    if err != nil {
        log.Fatalf("error: %v\n", err)

    }

    log.Printf("Started up interface: %s\n", intf.Name())

    sender := ipoe.NewDefaultSMTPSender()

    payload := make([]byte, 9000);
    
//    go ipoe.DoIMAP()

    for {
        n, err := intf.Read(payload)
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("Packet received of %d bytes\n", n)
        version := int(payload[0] >> 4)
        if version == 4 {

            packet, err := ipv4.ParseHeader(payload)
            if err != nil {
                log.Fatalf("Unable to parse packet: %v", err)
            }
            log.Printf("%s -> %s (%d)\n", packet.Src, packet.Dst, packet.Protocol)
            err = sender.SendV4(packet, payload)
            if err != nil {
                log.Fatalf("Unable to send mail: %v", err)
            }
        } else {
            packet, err := ipv6.ParseHeader(payload)
            if err != nil {
                log.Fatalf("Unable to parse packet: %v", err)
            }
            log.Printf("%s -> %s (%d)\n", packet.Src, packet.Dst, packet.NextHeader)
        }
    }


}
