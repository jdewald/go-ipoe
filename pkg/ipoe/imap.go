// vim: et:ts=4:sw=4
package ipoe

import (

    "github.com/emersion/go-imap/v2"
    "github.com/emersion/go-imap/v2/imapclient"

    "os"
    "log"
)

func DoIMAP() {
    c, err := imapclient.DialTLS("imap.gmail.com:993", &imapclient.Options { DebugWriter: os.Stdout })
	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	if err := c.Login("USERNAME", "PASSWORD").Wait(); err != nil {
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

	selectedMbox, err := c.Select("PAPERLESS", nil).Wait()
	if err != nil {
		log.Fatalf("failed to select PAPERLESS: %v", err)
	}
	log.Printf("PAPERLESS contains %v messages", selectedMbox.NumMessages)

	if selectedMbox.NumMessages > 0 {
		seqSet := imap.SeqSetNum(1)
		fetchOptions := &imap.FetchOptions{Envelope: true}
		messages, err := c.Fetch(seqSet, fetchOptions).Collect()
		if err != nil {
			log.Fatalf("failed to fetch first message in INBOX: %v", err)
		}
		log.Printf("subject of first message in PAPERLESS: %v", messages[0].Envelope.Subject)
	}

	if err := c.Logout().Wait(); err != nil {
		log.Fatalf("failed to logout: %v", err)
	}
}
