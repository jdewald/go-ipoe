// vim: et:ts=4:sw=4
package ipoe

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"golang.org/x/net/ipv4"
)

type SenderSMTPConfig struct {
	Username string
	Password string
	Server   string
	Email    string
	StartTLS bool
}

type DefaultSMTPSender struct {
	Config SenderSMTPConfig
	Router RoutingTable
	Codec
}

func NewDefaultSMTPSender(config SenderSMTPConfig, router RoutingTable, codec Codec) *DefaultSMTPSender {
	return &DefaultSMTPSender{Config: config, Router: router, Codec: codec}
}

func V4ToMail(from string, hdr *ipv4.Header, payload []byte, router RoutingTable, codec Codec) (*MailMessage, error) {

	// If we're in a "regime" where we are attempting to hide ourselves,
	// then we would not add these headers. BUt useful for debugging!

	message := NewMailMessage(from, fmt.Sprintf("%s->%s (%d)", hdr.Src, hdr.Dst, hdr.Protocol))
	// TODO: dec TTL?
	// To do that, need to update checksum

	// dest address will depend on the "ARP"/route lookup that maps dest IP to email
	to, err := router.Route(hdr.Dst.String())
	if err != nil {
		return nil, err
	}

	message.Recipient = append(message.Recipient, to)

	message.AddHeader("TTL", fmt.Sprintf("%d", hdr.TTL))
	message.AddHeader("SRC-IP", fmt.Sprintf("%s", hdr.Src))
	message.AddHeader("DEST-IP", fmt.Sprintf("%s", hdr.Dst))
	message.AddHeader("PROTO", fmt.Sprintf("%d", hdr.Protocol))
	message.AddHeader("TOTLEN", fmt.Sprintf("%d", hdr.TotalLen))

	encoded := codec.Encode(payload[0:hdr.TotalLen])
	message.Body = []byte(encoded)

	return message, nil

}

func (em *DefaultSMTPSender) Client() (*smtp.Client, error) {

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         em.Config.Server,
	}

	auth := smtp.PlainAuth("", em.Config.Username, em.Config.Password, em.Config.Server)

	var client *smtp.Client
	var err error

	if em.Config.StartTLS {

		client, err = smtp.Dial(fmt.Sprintf("%s:587", em.Config.Server))
		if err != nil {
			fmt.Println("Unable to dial!")
			return nil, err
		}

		err = client.StartTLS(tlsConfig)
		if err != nil {
			return nil, err
		}
	}

	err = client.Auth(auth)
	if err != nil {
		fmt.Println("Unable to Auth!")
		return client, err
	}

	return client, nil

}

// WriteTo will send a single packet over email.
//
// NOTE: The system may choose to coalesce multiple together
func (em *DefaultSMTPSender) SendV4(hdr *ipv4.Header, payload []byte) error {

	message, err := V4ToMail(em.Config.Email, hdr, payload, em.Router, em.Codec)
	if err != nil {
		return err
	}

	client, err := em.Client()
	if err != nil {
		return err
	}

	err = client.Mail(em.Config.Email)
	if err != nil {
		return fmt.Errorf("unable to initial mail from %s: %v", em.Config.Email, err)
	}

	for _, rcpt := range message.Recipient {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("unable to add %s as recipient: %v", rcpt, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(wc, message.ToData())
	if err != nil {
		return fmt.Errorf("unable to send complete data frame: %v", err)
	}

	wc.Close()

	err = client.Quit()
	if err != nil {
		return err
	}

	return nil
}
