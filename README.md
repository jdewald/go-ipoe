# IP-over-Email (IPoE)

Yes, exactly what it sounds like. Let's suppose you're in a heavily locked down environment where
the only external system you can talk to is an email provider. Or, you just want to do
something whacky.

This effectively creates a tunnel where the "Layer 2" is actual email.


## Architecture

Strictly speaking, sending and receiving are two completely separate systems, but usually
you would have both running together to create a bi-directional interface.


### Sending
To _send_ IP packets, we make use of a Linux `tun` interface which is set as the destination
interface for any destination networks intended to flow through this VPN. One easy way to do this
is to associate an an interface IP with it and have other end also be an IPoE system.

For example, you could do a 100.64.42.2/30 on one end and 100.64.42.3/30 on the other end.

The `tun` interface is in fact brought up by firing up the application that handles the send side,
configured with appropriate `SMTP` credentials to send emails, as well as the "next hops"
(email addresses) of any endpoints intended to be reached through through the tunnel.

### Receiving

To _receive_ IP packets, the application will make an `IMAP` connection (or connections) to
a server + mailbox where the packet emails will land. Upon receiving those emails, the packets
will be decoded and delivered to the local machine (to be routed/delivered as appropriately).
