package config

import (
	"bytes"
	. "launchpad.net/gocheck"
	"log"
	"os"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

func init() {
	setLogger() // This had to be done for DisplayErrors' test
}

func setLogger() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}
}

func (s *s) TestConfig(c *C) {
	config := CreateConfig()
	c.Assert(config.Servers, NotNil)
	c.Assert(config.Global, NotNil)
}

func (s *s) TestConfig_Fallbacks(c *C) {
	config := CreateConfig()

	host, name := "irc.gamesurge.net", "gamesurge"
	config.Global = &Server{
		config, "irc", "irc.nuclearfallout.net",
		5555, true, true, true, true, true, true, 500,
		"n1", "a1", "u1", "h1", "r1", "p1",
		[]string{"#chan", "#chan2"},
	}

	server := &Server{parent: config, Name: name, Host: host}
	config.Servers[name] = server

	c.Assert(server.GetHost(), Equals, host)
	c.Assert(server.GetName(), Equals, name)
	c.Assert(server.GetPort(), Equals, config.Global.Port)
	c.Assert(server.GetSsl(), Equals, config.Global.Ssl)
	c.Assert(server.GetVerifyCert(), Equals, config.Global.VerifyCert)
	c.Assert(server.GetNoReconnect(), Equals, config.Global.NoReconnect)
	c.Assert(server.GetReconnectTimeout(), Equals,
		config.Global.ReconnectTimeout)
	c.Assert(server.GetNick(), Equals, config.Global.Nick)
	c.Assert(server.GetAltnick(), Equals, config.Global.Altnick)
	c.Assert(server.GetUsername(), Equals, config.Global.Username)
	c.Assert(server.GetUserhost(), Equals, config.Global.Userhost)
	c.Assert(server.GetRealname(), Equals, config.Global.Realname)
	c.Assert(server.GetPrefix(), Equals, config.Global.Prefix)
	c.Assert(len(server.GetChannels()), Equals, len(config.Global.Channels))
	for i, v := range server.GetChannels() {
		c.Assert(v, Equals, config.Global.Channels[i])
	}

	//Check bools more throughly
	server.IsSslSet = true
	server.IsVerifyCertSet = true
	server.IsNoReconnectSet = true
	c.Assert(server.GetSsl(), Equals, false)
	c.Assert(server.GetVerifyCert(), Equals, false)
	c.Assert(server.GetNoReconnect(), Equals, false)

	server.IsSslSet = false
	server.IsVerifyCertSet = false
	server.IsNoReconnectSet = false
	config.Global.Ssl = false
	config.Global.VerifyCert = false
	config.Global.NoReconnect = false
	c.Assert(server.GetSsl(), Equals, false)
	c.Assert(server.GetVerifyCert(), Equals, false)
	c.Assert(server.GetNoReconnect(), Equals, false)

	//Check default values more thoroughly
	config.Global.Port = 0
	c.Assert(server.GetPort(), Equals, uint16(defaultIrcPort))
	config.Global.Prefix = ""
	c.Assert(server.GetPrefix(), Equals, ".")
	config.Global.ReconnectTimeout = 0
	c.Assert(server.GetReconnectTimeout(), Equals,
		uint(defaultReconnectTimeout))
}

func (s *s) TestConfig_Fluent(c *C) {
	srv1 := Server{
		nil, "irc", "irc.gamesurge.net",
		5555, true, true, false, true, false, true, 10,
		"n1", "a1", "u1", "h1", "r1", "p1",
		[]string{"#chan", "#chan2"},
	}
	defs := Server{
		nil, "irc2", "nuclearfallout.gamesurge.net",
		7777, false, false, true, false, false, false, 10,
		"n2", "a2", "u2", "h2", "r2", "p2",
		[]string{"#chan2"},
	}
	srv2 := "znc.gamesurge.net"

	conf := CreateConfig().
		Host(""). // Should not break anything
		Port(defs.Port).
		ReconnectTimeout(defs.ReconnectTimeout).
		Nick(defs.Nick).
		Altnick(defs.Altnick).
		Username(defs.Username).
		Userhost(defs.Userhost).
		Realname(defs.Realname).
		Prefix(defs.Prefix).
		Channels(defs.Channels...).
		Server(srv1.Name).
		Host(srv1.Host).
		Port(srv1.Port).
		Ssl(srv1.Ssl).
		VerifyCert(srv1.VerifyCert).
		NoReconnect(srv1.NoReconnect).
		ReconnectTimeout(srv1.ReconnectTimeout).
		Nick(srv1.Nick).
		Altnick(srv1.Altnick).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Realname(srv1.Realname).
		Prefix(srv1.Prefix).
		Channels(srv1.Channels...).
		Server(srv2)

	server := conf.Servers[srv1.Name]
	server2 := conf.Servers[srv2]
	c.Assert(server.GetHost(), Equals, srv1.GetHost())
	c.Assert(server.GetName(), Equals, srv1.GetName())
	c.Assert(server.GetPort(), Equals, srv1.GetPort())
	c.Assert(server.GetSsl(), Equals, srv1.GetSsl())
	c.Assert(server.GetVerifyCert(), Equals, srv1.GetVerifyCert())
	c.Assert(server.GetNoReconnect(), Equals, srv1.GetNoReconnect())
	c.Assert(server.GetReconnectTimeout(), Equals, srv1.GetReconnectTimeout())
	c.Assert(server.GetNick(), Equals, srv1.GetNick())
	c.Assert(server.GetAltnick(), Equals, srv1.GetAltnick())
	c.Assert(server.GetUsername(), Equals, srv1.GetUsername())
	c.Assert(server.GetUserhost(), Equals, srv1.GetUserhost())
	c.Assert(server.GetRealname(), Equals, srv1.GetRealname())
	c.Assert(server.GetPrefix(), Equals, srv1.GetPrefix())
	c.Assert(len(server.GetChannels()), Equals, len(srv1.GetChannels()))
	for i, v := range server.GetChannels() {
		c.Assert(v, Equals, srv1.Channels[i])
	}

	c.Assert(server2.GetHost(), Equals, srv2)
	c.Assert(server2.GetPort(), Equals, defs.GetPort())
	c.Assert(server2.GetSsl(), Equals, defs.GetSsl())
	c.Assert(server2.GetVerifyCert(), Equals, defs.GetVerifyCert())
	c.Assert(server2.GetNoReconnect(), Equals, defs.GetNoReconnect())
	c.Assert(server2.GetReconnectTimeout(), Equals, defs.GetReconnectTimeout())
	c.Assert(server2.GetNick(), Equals, defs.GetNick())
	c.Assert(server2.GetAltnick(), Equals, defs.GetAltnick())
	c.Assert(server2.GetUsername(), Equals, defs.GetUsername())
	c.Assert(server2.GetUserhost(), Equals, defs.GetUserhost())
	c.Assert(server2.GetRealname(), Equals, defs.GetRealname())
	c.Assert(server2.GetPrefix(), Equals, defs.GetPrefix())
	c.Assert(len(server2.GetChannels()), Equals, len(defs.GetChannels()))
	for i, v := range server2.GetChannels() {
		c.Assert(v, Equals, defs.Channels[i])
	}
}

func (s *s) TestConfig_Validation(c *C) {
	srv1 := Server{
		nil, "irc", "irc.gamesurge.net",
		5555, true, true, false, true, false, true, 10,
		"n1", "a1", "u1", "h1", "r1", "p1",
		[]string{"#chan", "#chan2"},
	}

	conf := CreateConfig()
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Not(Equals), 0)

	conf = CreateConfig().
		Server("").
		Port(srv1.Port)
	c.Assert(len(conf.Servers), Equals, 0)
	c.Assert(conf.Global.Port, Equals, uint16(srv1.Port))
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Equals, 2)

	conf = CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server("a.com").
		Server("a.com")
	c.Assert(len(conf.Servers), Equals, 1)
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Equals, 1)

	conf = CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server("%")
	c.Assert(conf.IsValid(), Equals, false)
	// Invalid: Host
	c.Assert(len(conf.Errors), Equals, 1)

	conf = CreateConfig().
		Server(srv1.Host)
	c.Assert(conf.IsValid(), Equals, false)
	// Missing: Nick, Realname, Username, Userhost
	c.Assert(len(conf.Errors), Equals, 4)

	conf = CreateConfig().
		Server(srv1.Host).
		Nick(`@Nick`).              // no special chars
		Channels(`chan`).           // must start with valid prefix
		Username(`spaces in here`). // no spaces
		Userhost(`~#@#$@!`).        // must be a host
		Realname(`@ !`)             // no special chars
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Equals, 5)

	conf = CreateConfig().
		Server(srv1.Host).
		Nick(srv1.Nick).
		Channels(srv1.Channels...).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Realname(srv1.Realname)
	conf.Servers[srv1.Host].Host = ""
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Equals, 1) // No host
	conf.Errors = nil
	conf.Servers[srv1.Host].Host = "@@@"
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Equals, 1) // Bad host
}

func (s *s) TestConfig_DisplayErrors(c *C) {
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	c.Assert(buf.Len(), Equals, 0)
	conf := CreateConfig().
		Server("localhost")
	c.Assert(conf.IsValid(), Equals, false)
	c.Assert(len(conf.Errors), Equals, 4)
	conf.DisplayErrors()
	c.Assert(buf.Len(), Not(Equals), 0)
	setLogger() // Reset the logger
}

func (s *s) TestValidNames(c *C) {
	goodNicks := []string{`a1bc`, `a5bc`, `a9bc`, `MyNick`, `[MyNick`,
		`My[Nick`, `]MyNick`, `My]Nick`, `\MyNick`, `My\Nick`, "MyNick",
		"My`Nick", `_MyNick`, `My_Nick`, `^MyNick`, `My^Nick`, `{MyNick`,
		`My{Nick`, `|MyNick`, `My|Nick`, `}MyNick`, `My}Nick`,
	}

	badNicks := []string{`My Name`, `My!Nick`, `My"Nick`, `My#Nick`, `My$Nick`,
		`My%Nick`, `My&Nick`, `My'Nick`, `My/Nick`, `My(Nick`, `My)Nick`,
		`My*Nick`, `My+Nick`, `My,Nick`, `My-Nick`, `My.Nick`, `My/Nick`,
		`My;Nick`, `My:Nick`, `My<Nick`, `My=Nick`, `My>Nick`, `My?Nick`,
		`My@Nick`, `1abc`, `5abc`, `9abc`, `@ChanServ`,
	}

	for i := 0; i < len(goodNicks); i++ {
		if !rgxNickname.MatchString(goodNicks[i]) {
			c.Errorf("Good nick failed regex: %v\n", goodNicks[i])
		}
	}
	for i := 0; i < len(badNicks); i++ {
		if rgxNickname.MatchString(badNicks[i]) {
			c.Errorf("Bad nick passed regex: %v\n", badNicks[i])
		}
	}
}

func (s *s) TestValidChannels(c *C) {
	// Check that the first letter must be {#+!&}
	goodChannels := []string{"#ValidChannel", "+ValidChannel", "&ValidChannel",
		"!12345", "#c++"}

	badChannels := []string{"#Invalid Channel", "#Invalid,Channel",
		"#Invalid\aChannel", "#", "+", "&", "InvalidChannel"}

	for i := 0; i < len(goodChannels); i++ {
		if !rgxChannel.MatchString(goodChannels[i]) {
			c.Errorf("Good chan failed regex: %v\n", goodChannels[i])
		}
	}
	for i := 0; i < len(badChannels); i++ {
		if rgxChannel.MatchString(badChannels[i]) {
			c.Errorf("Bad chan passed regex: %v\n", badChannels[i])
		}
	}
}
