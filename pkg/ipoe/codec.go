// vim: et:ts=4:sw=4
package ipoe

import (
    "fmt"
    "strings"
)

const (
    HeaderPrefix = "X-IPOE-"
    StartDelimiter = "--- START IPOE ---\r\n"
    EndDelimiter = "--- END IPOE --\r\n"
)
type Codec interface {

}

type MailMessage struct {
    Subject string
    From string
    Recipient []string
    Body []byte
    Headers map[string]string
}

func NewMailMessage(from, subject string) *MailMessage {
    return &MailMessage{
        Subject: subject,
        From: from,
        Recipient: []string{},
        Body: nil,
        Headers: map[string]string{},
    }
}

func (mm *MailMessage) AddHeader(hdr, value string) *MailMessage {
    mm.Headers[fmt.Sprintf("%s%s", HeaderPrefix, hdr)] = value
    return mm
}

func (mm *MailMessage) ToData() string{
    var builder strings.Builder

    builder.WriteString("Subject: ")
    builder.WriteString(mm.Subject)
    builder.WriteString("\r\n")
    // TODO: BCC this?
    builder.WriteString("To: ")
    builder.WriteString(mm.Recipient[0])
    builder.WriteString("\r\n")
    for k,v := range mm.Headers {
        builder.WriteString(k)
        builder.WriteString(": ")
        builder.WriteString(v)
        builder.WriteString("\r\n")
    }
    builder.WriteString("\r\n")
    builder.WriteString(StartDelimiter)
    builder.Write(mm.Body)
    builder.WriteString("\r\n")
    builder.WriteString(EndDelimiter)

    // We may want to BCC to actual mail recipients
    data := builder.String()
    
    fmt.Printf("DATA: %s", data)
    return data
}
