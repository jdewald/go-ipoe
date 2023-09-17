// vim: et:ts=4:sw=4
package ipoe

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"golang.org/x/net/ipv4"

	"log"
	"os"
)

type IMAPConfig struct {
	Server   string
	Username string
	Password string
	Mailbox  string
}

type IMAPReceiver struct {
	Config IMAPConfig
	Codec  Codec
}

func (ir *IMAPReceiver) Listen(ctx context.Context) {
	config := ir.Config

	haveMessages := make(chan uint32, 1)

	options := imapclient.Options{
		DebugWriter: os.Stdout,
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Expunge: func(seqNum uint32) {
				log.Printf("message %v has been expunged", seqNum)
			},
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					log.Printf("%d new messages received\n", *data.NumMessages)
					haveMessages <- *data.NumMessages
				}
			},
			Fetch: func(msg *imapclient.FetchMessageData) {
				log.Printf("Received FETCH data")
			},
		},
	}
	c, err := imapclient.DialTLS(config.Server, &options)

	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Logout()
	defer c.Close()

	if err := c.Login(config.Username, config.Password).Wait(); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	mailboxes, err := c.List("", "%", nil).Collect()
	if err != nil {
		log.Fatalf("failed to list mailboxes: %v", err)
	}
	log.Printf("Found %v mailboxes", len(mailboxes))
	for _, mbox := range mailboxes {
		log.Printf(" - %v", mbox.Mailbox)
	}

	selectedMbox, err := c.Select(config.Mailbox, nil).Wait()
	if err != nil {
		log.Fatalf("failed to select %s: %v", config.Mailbox, err)
	}
	log.Printf("%s contains %v messages", config.Mailbox, selectedMbox.NumMessages)

	// https://datatracker.ietf.org/doc/html/rfc2177
	idleCommand, err := c.Idle()
	if err != nil {
		log.Fatalf("server doesn't support IDLE, will need to poll: %v", err)
	}

	// ipv4
	//	ipv4fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	for {

		select {
		case <-ctx.Done():
			close(haveMessages)
			idleCommand.Close()
		case num, havemore := <-haveMessages:
			fmt.Printf("Received notification of %d messages (%v)\n", num, havemore)
			idleCommand.Close()
			if !havemore {
				break
			}

			if num > 0 {
				seqSet := imap.SeqSetRange(0, num)
				fetchOptions := &imap.FetchOptions{
					Envelope: true,
					BodySection: []*imap.FetchItemBodySection{
						{
							Specifier: imap.PartSpecifierText,
						},
					},
				}
				messages, err := c.Fetch(seqSet, fetchOptions).Collect()
				if err != nil {
					log.Fatalf("failed to fetch first message in %s: %v", config.Mailbox, err)
				}

				for i, message := range messages {
					log.Printf("subject of %d message in %s: %v", i, config.Mailbox, message.Envelope.Subject)

					for sect, data := range message.BodySection {
						fmt.Printf("Data from %s\n", sect.Specifier)
						strData := string(data)
						if strings.Contains(strData, StartDelimiter) {
							fmt.Println("Found packet data")
							encIndex := strings.Index(strData, StartDelimiter)
							encIndex += len(StartDelimiter)
							lastIndex := strings.Index(strData, EndDelimiter)

							encoded := strData[encIndex:lastIndex]

							decoded, err := ir.Codec.Decode(encoded)
							if err != nil {
								fmt.Printf("Error decoded packet: %v", err)
							} else {
								if hdr, err := ipv4.ParseHeader(decoded); err == nil {
									//									addr := syscall.SockaddrInet4{
									//										Port: 0,
									//										Addr: [4]byte{hdr.Dst[0], hdr.Dst[1], hdr.Dst[2], hdr.Dst[3]},
									//									}

									if conn, err := net.Dial(fmt.Sprintf("ip:%d", hdr.Protocol), hdr.Dst.String()); err == nil {
										raw, err := ipv4.NewRawConn(conn.(net.PacketConn))
										if err != nil {
											log.Fatalf("Unable to create Raw conn: %v", err)
										}
										if _, writeErr := raw.Write(decoded); writeErr != nil {
											log.Fatalf("Unable to deliver IP packet: %v", writeErr)
										}
										conn.Close()
									} else {
										log.Fatalf("Unable to dial to %s:%v", hdr.Dst, err)
									}
									//									sendErr := syscall.Sendto(ipv4fd, decoded, 0, &addr)
									//									if sendErr != nil {
									//										log.Fatalf("Unable to send packet to %s:%v\n", hdr.Dst, sendErr)
									//								}
								} else {
									fmt.Printf("Unable to parse IP packet: %v", err)
								}
							}

						}

					}

				}
				_, err = c.UIDMove(seqSet, "IPOE-Processed").Wait()
				if err != nil {
					if cErr := c.Create("IPOE-Processed", &imap.CreateOptions{}).Wait(); cErr == nil {
						if _, err = c.UIDMove(seqSet, "IPOE-Processed").Wait(); err != nil {
							log.Fatalf("Unable to move messages: %v", err)
						}
					}
				}

			}
			idleCommand, err = c.Idle()
			if err != nil {
				log.Fatalf("server doesn't support IDLE, will need to poll: %v", err)
			}
		}

	}

}
