package bot

import (
	"bytes"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"net"
	. "testing"
)

var zeroConnProvider = func(srv string) (net.Conn, error) {
	return nil, nil
}

func TestBotConfig_ReadConfig(t *T) {
	b, _ := createBot(fakeConfig, nil, nil, false, false)

	b.ReadConfig(func(conf *config.Config) {
		if conf.Servers[serverID].GetNick() !=
			fakeConfig.Servers[serverID].GetNick() {

			t.Error("The names should have been the same.")
		}
	})
}

func TestBotConfig_WriteConfig(t *T) {
	b, _ := createBot(fakeConfig, nil, nil, false, false)

	b.WriteConfig(func(conf *config.Config) {
		if conf.Servers[serverID].GetNick() !=
			fakeConfig.Servers[serverID].GetNick() {

			t.Error("The names should have been the same.")
		}
	})
}

func TestBotConfig_testElementEquals(t *T) {
	a := []string{"a", "b"}
	b := []string{"b", "a"}
	if !contains(a, b) {
		t.Error("Expected equals.")
	}

	a = []string{"a", "b", "c"}
	if contains(a, b) {
		t.Error("Expected not equals.")
	}

	a = []string{"x", "y"}
	if contains(a, b) {
		t.Error("Expected not equals.")
	}

	a = []string{}
	b = []string{}
	if !contains(a, b) {
		t.Error("Expected equals.")
	}

	b = []string{"a"}
	if contains(a, b) {
		t.Error("Expected not equals.")
	}

	a = []string{"a"}
	b = []string{}
	if contains(a, b) {
		t.Error("Expected not equals.")
	}
}

func TestBotConfig_ReplaceConfig(t *T) {
	nick := []byte(irc.NICK + " :newnick\r\n")

	conns := map[string]*mocks.Conn{
		serverID + ":6667":      mocks.CreateConn(),
		"newserver:6667":        mocks.CreateConn(),
		"anothernewserver:6667": mocks.CreateConn(),
	}
	connProvider := func(srv string) (net.Conn, error) {
		c := conns[srv]
		if c == nil {
			panic("No connection found:" + srv)
		}
		return conns[srv], nil
	}

	chans1 := []string{"#chan1", "#chan2", "#chan3"}
	chans2 := []string{"#chan1", "#chan3"}
	chans3 := []string{"#chan1"}

	c1 := fakeConfig.Clone().
		GlobalContext().
		Channels(chans1...).
		Server("newserver")

	c2 := fakeConfig.Clone().
		GlobalContext().
		Channels(chans2...).
		ServerContext(serverID).
		Nick("newnick").
		Channels(chans3...).
		Server("anothernewserver")

	c3 := &config.Config{}

	b, _ := createBot(c1, connProvider, nil, false, false)
	if len(c1.Servers) != len(b.servers) {
		t.Errorf("The number of servers (%v) should match the config (%v)",
			len(b.servers), len(c1.Servers))
	}

	oldsrv1, oldsrv2 := b.servers[serverID], b.servers["newserver"]
	old1listen, old2listen := make(chan Status), make(chan Status)

	oldsrv1.addStatusListener(old1listen, STATUS_STARTED)
	oldsrv2.addStatusListener(old2listen, STATUS_STARTED)

	end := b.Start()

	<-old1listen
	<-old2listen

	if e := b.conf.Global.Channels; !contains(e, chans1) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := oldsrv1.conf.GetChannels(); !contains(e, chans1) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := oldsrv2.conf.GetChannels(); !contains(e, chans1) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := b.dispatchCore.GetChannels(); !contains(e, chans1) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := oldsrv1.dispatchCore.GetChannels(); !contains(e, chans1) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := oldsrv2.dispatchCore.GetChannels(); !contains(e, chans1) {
		t.Errorf("Expected elements: %v", e)
	}

	success := b.ReplaceConfig(c3) // Invalid Config
	if success {
		t.Error("An invalid config should fail.")
	}
	success = b.ReplaceConfig(c2)
	if !success {
		t.Error("A valid new config should succeed.")
	}

	if <-end == nil {
		t.Error("Expected a kill error")
	}

	newsrv1 := b.servers["anothernewserver"]

	if len(c2.Servers) != len(b.servers) {
		t.Errorf("The number of servers (%v) should match the config (%v)",
			len(b.servers), len(c2.Servers))
	}

	if e := b.conf.Global.Channels; !contains(e, chans2) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := oldsrv1.conf.GetChannels(); !contains(e, chans3) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := newsrv1.conf.GetChannels(); !contains(e, chans2) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := b.dispatchCore.GetChannels(); !contains(e, chans2) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := oldsrv1.dispatchCore.GetChannels(); !contains(e, chans3) {
		t.Errorf("Expected elements: %v", e)
	}
	if e := newsrv1.dispatchCore.GetChannels(); !contains(e, chans2) {
		t.Errorf("Expected elements: %v", e)
	}

	recv := conns[serverID+":6667"].Receive(len(nick), nil)
	if bytes.Compare(recv, nick) != 0 {
		t.Errorf("Was expecting a change in nick but got: %s", recv)
	}

	b.Stop()
	for _ = range end {
	}
}
