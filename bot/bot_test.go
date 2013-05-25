package bot

import (
	"code.google.com/p/gomock/gomock"
	"github.com/aarondl/ultimateq/config"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"github.com/aarondl/ultimateq/irc"
	"io"
	. "launchpad.net/gocheck"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

type testHandler struct {
	callback func(*irc.IrcMessage, irc.Sender)
}

func (h testHandler) HandleRaw(m *irc.IrcMessage, send irc.Sender) {
	if h.callback != nil {
		h.callback(m, send)
	}
}

func init() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}
}

var serverId = "irc.gamesurge.net"

var fakeConfig = Configure().
	Nick("nobody").
	Altnick("nobody1").
	Username("nobody").
	Userhost("bitforge.ca").
	Realname("ultimateq").
	NoReconnect(true).
	Ssl(true).
	Server(serverId)

//==================================
// Tests begin
//==================================
func (s *s) TestCreateBot(c *C) {
	bot, err := CreateBot(fakeConfig)
	c.Assert(bot, NotNil)
	c.Assert(err, IsNil)
	_, err = CreateBot(Configure())
	c.Assert(err, Equals, errInvalidConfig)
	_, err = CreateBot(ConfigureFunction(
		func(conf *config.Config) *config.Config {
			return fakeConfig
		}),
	)
	c.Assert(err, IsNil)
}

func (s *s) TestBot_StartStop(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Read(gomock.Any()).Return(0, net.ErrWriteToConnected)
	conn.EXPECT().Close()

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	srv := b.servers[serverId]
	srv.dispatcher.Unregister(irc.RAW, srv.handlerId)
	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.Start()
	b.Stop()
	b.Disconnect()
	b.WaitForHalt()
}

func (s *s) TestBot_StartStopServer(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Read(gomock.Any()).Return(0, net.ErrWriteToConnected)
	conn.EXPECT().Close()
	conn.EXPECT().Close()

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)

	srv := b.servers[serverId]
	srv.dispatcher.Unregister(irc.RAW, srv.handlerId)
	c.Assert(srv.IsStarted(), Equals, false)
	c.Assert(srv.IsConnected(), Equals, false)

	_, err = b.ConnectServer(serverId)
	c.Assert(err, IsNil)
	c.Assert(srv.IsConnected(), Equals, true)

	_, err = b.ConnectServer(serverId)
	c.Assert(err, NotNil)

	b.StartServer(serverId)
	c.Assert(srv.IsStarted(), Equals, true)

	b.StopServer(serverId)
	c.Assert(srv.IsStarted(), Equals, false)

	b.DisconnectServer(serverId)
	c.Assert(srv.IsConnected(), Equals, false)

	b.WaitForHalt()

	_, err = b.ConnectServer(serverId)
	c.Assert(err, IsNil)
	b.DisconnectServer(serverId)
}

func (s *s) TestBot_Reconnecting(c *C) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(false).
		ReconnectTimeout(1).Ssl(true).Server(serverId)

	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
	conn.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
	conn.EXPECT().Close()
	conn.EXPECT().Close()

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(conf, nil, connProvider)
	c.Assert(err, IsNil)
	srv := b.servers[serverId]
	srv.reconnScale = time.Millisecond

	mu := sync.Mutex{}
	discs := 0

	handler := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		if m.Name == irc.DISCONNECT {
			mu.Lock()
			discs++
			if discs >= 2 {
				b.InterruptReconnect(serverId)
				b.disconnectServer(srv)
			}
			mu.Unlock()
		}
	}}

	srv.dispatcher.Unregister(irc.RAW, srv.handlerId)
	b.Register(irc.DISCONNECT, handler)
	b.Connect()
	b.start(false, true)
	b.WaitForHalt()
}

func (s *s) TestBot_Dispatching(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	str := []byte("PRIVMSG #chan :msg\r\n#\r\n")

	conn := mocks.NewMockConn(mockCtrl)
	mocks.ByteFiller = str
	conn.EXPECT().Read(gomock.Any()).Return(len(str), io.EOF)
	conn.EXPECT().Close()

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	waiter := sync.WaitGroup{}
	waiter.Add(1)
	b, err := createBot(fakeConfig, nil, connProvider)
	srv := b.servers[serverId]
	srv.dispatcher.Unregister(irc.RAW, srv.handlerId)

	b.Register(irc.PRIVMSG, &testHandler{
		func(m *irc.IrcMessage, send irc.Sender) {
			waiter.Done()
		},
	})

	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.start(false, true)
	waiter.Wait()
	b.Stop()
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestBot_Register(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	gid := b.Register(irc.PRIVMSG, &coreHandler{})
	id, err := b.RegisterServer(serverId, irc.PRIVMSG, &coreHandler{})
	c.Assert(err, IsNil)

	c.Assert(b.Unregister(irc.PRIVMSG, id), Equals, false)
	c.Assert(b.Unregister(irc.PRIVMSG, gid), Equals, true)

	ok, err := b.UnregisterServer(serverId, irc.PRIVMSG, gid)
	c.Assert(ok, Equals, false)
	ok, err = b.UnregisterServer(serverId, irc.PRIVMSG, id)
	c.Assert(ok, Equals, true)

	_, err = b.RegisterServer("", "", &coreHandler{})
	c.Assert(err, Equals, errUnknownServerId)
	_, err = b.UnregisterServer("", "", 0)
	c.Assert(err, Equals, errUnknownServerId)
}

func (s *s) TestcreateBot(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	capsProvider := func() *irc.ProtoCaps {
		return irc.CreateProtoCaps()
	}
	connProvider := func(srv string) (net.Conn, error) {
		return mocks.NewMockConn(mockCtrl), nil
	}

	b, err := createBot(fakeConfig, capsProvider, connProvider)
	c.Assert(b, NotNil)
	c.Assert(err, IsNil)
	c.Assert(len(b.servers), Equals, 1)
	c.Assert(b.caps, NotNil)
	c.Assert(b.capsProvider, NotNil)
	c.Assert(b.connProvider, NotNil)
}

func (s *s) TestBot_Providers(c *C) {
	capsProv := func() *irc.ProtoCaps {
		p := irc.CreateProtoCaps()
		p.ParseProtoCaps(&irc.IrcMessage{Args: []string{"nick", "CHANTYPES=H"}})
		return p
	}
	connProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}

	b, err := createBot(fakeConfig, capsProv, connProv)
	c.Assert(err, NotNil)
	c.Assert(err, Not(Equals), net.ErrWriteToConnected)
	b, err = createBot(fakeConfig, nil, connProv)
	ers := b.Connect()
	c.Assert(ers[0], Equals, net.ErrWriteToConnected)
}

func (s *s) TestBot_createIrcClient(c *C) {
	b, err := createBot(fakeConfig, nil, nil)
	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(ers[0], Equals, errSslNotImplemented)
}

func (s *s) TestBot_createDispatcher(c *C) {
	_, err := createBot(fakeConfig, func() *irc.ProtoCaps {
		return nil
	}, nil)
	c.Assert(err, NotNil)
}

func (s *s) TestServerSender(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	str := "PRIVMSG user :msg\r\n"

	conn := mocks.NewMockConn(mockCtrl)
	conn.Writechan = make(chan []byte)
	gomock.InOrder(
		conn.EXPECT().Write([]byte(str)).Return(len(str), nil),
		conn.EXPECT().Close(),
	)

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)
	srv := b.servers[serverId]
	srv.dispatcher.Unregister(irc.RAW, srv.handlerId)
	srvsender := ServerSender{serverId, srv}
	c.Assert(srvsender.GetKey(), Equals, serverId)

	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.start(true, false)
	srvsender.Writeln(str)
	<-conn.Writechan
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_State(c *C) {
	srv := &Server{}
	srv.setStarted(false)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setStopped(false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setStarted(true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setStopped(true)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setConnected(true)
	c.Assert(srv.IsConnected(), Equals, true)
	srv.setDisconnected(true)
	c.Assert(srv.IsConnected(), Equals, false)

	srv.setConnected(true)
	c.Assert(srv.IsConnected(), Equals, true)
	srv.setDisconnected(true)
	c.Assert(srv.IsConnected(), Equals, false)

	srv.setReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, true)
	srv.setNotReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, false)

	srv.setReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, true)
	srv.setNotReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, false)
}
