// vim: et:ts=4:sw=4
package ipoe

import (
    "crypto/tls"
	"net"
    "fmt"
	"net/smtp"
    "golang.org/x/net/ipv4"

	b64 "encoding/base64"
)

type DefaultSMTPSender struct {
    SourceAddress string
}

func NewDefaultSMTPSender() *DefaultSMTPSender{
    return &DefaultSMTPSender{
        SourceAddress: "SOURCE-ADDRESS"
    }
}


func Route(dest net.IP) (string, error) {
    return fmt.Sprintf("SOURCE-ADDRESS%s@gmail.com", dest), nil
}

func V4ToMail(from string, hdr *ipv4.Header, payload []byte) (*MailMessage, error) {

    // If we're in a "regime" where we are attempting to hide ourselves,
    // then we would not add these headers. BUt useful for debugging!

    message := NewMailMessage(from, fmt.Sprintf("%s->%s (%d)", hdr.Src, hdr.Dst, hdr.Protocol))
    // TODO: dec TTL?
    // To do that, need to update checksum

    // dest address will depend on the "ARP"/route lookup that maps dest IP to email
    to, err := Route(hdr.Dst)
    if err != nil {
        return nil, err
    }

    message.Recipient = append(message.Recipient, to)

    message.AddHeader("TTL", fmt.Sprintf("%d", hdr.TTL))
    message.AddHeader("SRC-IP", fmt.Sprintf("%s", hdr.Src))
    message.AddHeader("DEST-IP", fmt.Sprintf("%s", hdr.Dst))
    message.AddHeader("PROTO", fmt.Sprintf("%d",  hdr.Protocol))
    message.AddHeader("TOTLEN", fmt.Sprintf("%d", hdr.TotalLen))

    encoded := b64.URLEncoding.EncodeToString(payload[0:hdr.TotalLen])
    message.Body = []byte(encoded)

    return message, nil

}

func (em *DefaultSMTPSender) Client() (*smtp.Client, error) { 

    tlsConfig := &tls.Config {
        InsecureSkipVerify: false,
        ServerName: "smtp.gmail.com",
    }

    auth := smtp.PlainAuth("", "USERNAME", "PASSWORD", "smtp.gmail.com")

    client, err := smtp.Dial("smtp.gmail.com:587")
    if err != nil {
        fmt.Println("Unable to dial!")
        return nil, err
    }

/*    client, err := smtp.NewClient(conn, "smtp.gmail.com:587")
    if err != nil {
        return nil, err
    }
    */

    err = client.StartTLS(tlsConfig)
    if err != nil {
        return nil, err
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

    message, err := V4ToMail(em.SourceAddress, hdr, payload)
    if err != nil {
        return err
    }

    client, err := em.Client()
    if err != nil {
        return err
    }
    
    err = client.Mail(em.SourceAddress)
    if err != nil {
        return err
    }

    for _, rcpt := range message.Recipient {
        if err := client.Rcpt(rcpt); err != nil {
            return err
        }
    }

    wc, err := client.Data()
    if err != nil {
        return err
    }


    _, err = fmt.Fprintf(wc, message.ToData())
    if err != nil {
        return nil
    }

    wc.Close()

    err = client.Quit()
    if err != nil {
        return err
    }

    return nil
}

