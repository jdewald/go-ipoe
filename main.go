// vim: et:ts=4:sw=4
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jdewald/ipoveremail/pkg/ipoe"
	"github.com/songgao/water"
	"github.com/spf13/viper"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	DefaultInterfaceName = "ipoveremail0"
)

func init() {

	viper.SetEnvPrefix("ipoe")
	viper.AutomaticEnv()
	viper.SetDefault("interfacename", DefaultInterfaceName)
	viper.SetDefault("smtp.starttls", true)

}

func BindInterface(name string) *water.Interface {
	config := water.Config{
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

	return intf

}

func ListenPackets(ctx context.Context, intf *water.Interface, sender *ipoe.DefaultSMTPSender) {

	_payload := make([]byte, 9000)
	for {
		n, err := intf.Read(_payload)

		payload := _payload[:n]
		if intf.IsTAP() {
			payload = payload[14:] // ethernet without q-in-q

		}
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Packet received of %d/%d bytes\n", len(payload), n)
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
				log.Fatalf("Unable to parse V6 packet: %v", err)
			}
			log.Printf("%s -> %s (%d)\n", packet.Src, packet.Dst, packet.NextHeader)
		}
	}

}

func ListenRemote(ctx context.Context, receiver ipoe.IPOEReciver, intf *water.Interface) {
	receiver.Listen(ctx, intf)

}
func main() {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool, 1)

	ctx, cancel := context.WithCancel(context.Background())

	// Stop if we get either signal, handling HUP
	// would allow reload
	go func() {
		<-sigCh
		cancel()
		done <- true
	}()

	viper.SetConfigName("ipoe-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/ipoe")
	viper.AddConfigPath(".")

	viper.ReadInConfig()

	smtpUser := viper.GetString("smtp.username")
	smtpPass := viper.GetString("smtp.password")
	smtpServer := viper.GetString("smtp.server")
	smtpStartTLS := viper.GetBool("smtp.starttls")

	destEmail := viper.GetString("routing.destemail")
	fromEmail := viper.GetString("routing.fromemail")

	imapUser := viper.GetString("imap.username")
	imapPass := viper.GetString("imap.password")
	imapServer := viper.GetString("imap.server")
	imapMailbox := viper.GetString("imap.mailbox")

	senderConfig := ipoe.SenderSMTPConfig{
		Username: smtpUser,
		Password: smtpPass,
		Server:   smtpServer,
		StartTLS: smtpStartTLS,
		Email:    fromEmail,
	}

	receiverConfig := ipoe.IMAPConfig{
		Username: imapUser,
		Password: imapPass,
		Server:   imapServer,
		Mailbox:  imapMailbox,
	}

	router := &ipoe.SameEmailRouting{DestEmail: destEmail}
	fmt.Printf("Routing via: %s\n", destEmail)

	b64Codec := &ipoe.Base64Codec{}
	sender := ipoe.NewDefaultSMTPSender(senderConfig, router, b64Codec)
	fmt.Printf("Sending from %s\n", senderConfig.Username)

	intfName := viper.GetString("interfacename")

	intf := BindInterface(intfName)

	// Packets we need to send
	go ListenPackets(ctx, intf, sender)

	// Packets we receive from remote. Note
	// that this is essentially unrelated to the send side
	go ListenRemote(ctx, &ipoe.IMAPReceiver{Config: receiverConfig, Codec: b64Codec}, intf)

	<-done
	fmt.Println("Goodbye!")

}
