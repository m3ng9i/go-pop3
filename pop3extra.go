// This file contains extra code based on https://github.com/bytbox/go-pop3
// Reference material: https://tools.ietf.org/html/rfc1939
package pop3

import (
    "crypto/tls"
    "fmt"
    "strconv"
    "strings"

    "github.com/m3ng9i/parsemail"
)

// DialTLSSkipVerify creates a TLS-secured connection to the POP3 server
// without certificate verification.
func DialTLSSkipVerify(addr string) (*Client, error) {
    var config  = tls.Config {
        InsecureSkipVerify : true,
    }

    return DialTLSWithConfig(addr, &config)
}


// DialTLSWithConfig creates a TLS-secured connection to the POP3 server. The
// param tlsConfig can be used for more sophisticated control about TLS
// transmission.
func DialTLSWithConfig(addr string, tlsConfig *tls.Config) (*Client, error) {
    conn, err := tls.Dial("tcp", addr, tlsConfig)
    if err != nil {
        return nil, err
    }
    return NewClient(conn)
}


// UIDL returns the unique id of the given message, if it exists. If the message
// does not exist, or another error is encountered, the returned unique id will
// be "". Param msg means message number.
func (c *Client) UIDL(msg int) (uid string, err error) {
    l, err := c.Cmd("UIDL %d\r\n", msg)
    if err != nil {
        return
    }
    uid = strings.Fields(l)[1]
    return
}


// UidlAll returns a list of all message numbers and their unique ids.
func (c *Client) UidlAll() (msgs []int, uids []string, err error) {
    _, err = c.Cmd("UIDL\r\n")
    if err != nil {
        return
    }
    lines, err := c.ReadLines()
    if err != nil {
        return
    }
    msgs = make([]int, len(lines), len(lines))
    uids = make([]string, len(lines), len(lines))
    for i, l := range lines {
        var m int
        fs := strings.Fields(l)
        m, err = strconv.Atoi(fs[0])
        if err != nil {
            return
        }
        msgs[i] = m
        uids[i] = fs[1]
    }
    return
}


// TOP returns first n rows of a message.
func (c *Client) TOP(msg, n int) (text string, err error) {
    _, err = c.Cmd("TOP %d %d\r\n", msg, n)
    if err != nil {
        return
    }
    lines, err := c.ReadLines()
    if err != nil {
        return
    }
    text = strings.Join(lines, "\n")
    return
}


// GetMail get a mail by message number.
func (c *Client) GetMail(msg int) (email parsemail.Email, err error) {
    text, err := c.RETR(msg)
    if err != nil {
        return
    }

    email, err = parsemail.Parse(strings.NewReader(text))
    return
}


type MailItem struct {
    parsemail.Email
    Size    int
    MsgNum  int // message number
}


// Get basic mail info by message number. In the return value of email, not all fields are valid.
func (c *Client) GetInfo(msg int) (email parsemail.Email, err error) {
    text, err := c.TOP(msg, 120)
    if err != nil {
        return
    }

    email, err = parsemail.ParseHeader(strings.NewReader(text))
    return
}


// Get recent n's email item from the mailbox, if n <= 0, get all the email item.
// The most recent email item is in the front of the list slice.
func (c *Client) GetList(n int) (list []MailItem, err error) {
    msgs, sizes, err := c.ListAll()
    if err != nil {
        return
    }

    num := len(msgs)
    if num != len(sizes) {
        err = fmt.Errorf("GetList(): length of msgs and sizes are not the same.")
        return
    }

    for i := num - 1; i >= 0; i-- {
        var item = MailItem {
            Size    : sizes[i],
            MsgNum  : msgs[i],
        }

        list = append(list, item)

        // Get at most n's email list item
        if n > 0 && num - i >= n {
            break
        }
    }

    for i := 0; i < len(list); i++ {
        email, e := c.GetInfo(list[i].MsgNum)
        if e != nil {
            err = e
            return
        }

        list[i].Email = email
    }

    return
}

