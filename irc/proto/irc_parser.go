// Deals with irc protocol parsing
package proto

import (
	"errors"
	"strings"
)

var (
	// errHandlerNotRegistered: Handler not registered.
	errHandlerNotRegistered = errors.New("irc: Handler not registered")
	// errHandlerAlreadyRegistered: Handler previously registered.
	errHandlerAlreadyRegistered = errors.New("irc: Handler already registered")
	// errNoProtocolGiven: Happens when an empty string is given to parse
	errNoProtocolGiven = errors.New("irc: No protocol given")
	// errArgsAfterFinalNoColon: Colon not given, but arguments still followed.
	errArgsAfterFinalNoColon = errors.New(
		"irc: Colon not given, but arguments still followed")
	// errExpectedMoreArguments: Protocol line ends abruptly
	errExpectedMoreArguments = errors.New("irc: Expected more arguments")
)

// ParseResult: The result of a pass through the parsing tree.
type ParseResult struct {
	// The name of the event
	Name string
	// args: arguments in plain form
	Args map[string]string
	// nargs: split arguments
	Argv map[string][]string
	// channels: any channels found
	Channels map[string][]string
}

// IrcParser: Handles parsing irc and returning a result on how to handle.
type IrcParser struct {
	handlers map[string]*fragment
}

// CreateIrcParser: Creates an irc parser struct.
func CreateIrcParser() *IrcParser {
	return &IrcParser{handlers: make(map[string]*fragment)}
}

// createParseResult: Creates a parse result struct.
func createParseResult() *ParseResult {
	return &ParseResult{
		Args:     make(map[string]string),
		Argv:     make(map[string][]string),
		Channels: make(map[string][]string),
	}
}

// addHandler: Adds a handler to the IrcParser.
func (p *IrcParser) AddIrcHandler(handler string, tree *fragment) error {
	handler = strings.ToLower(handler)
	_, has := p.handlers[handler]
	if has {
		return errHandlerAlreadyRegistered
	}

	p.handlers[handler] = tree
	return nil
}

// removeHandler: Deletes a handler from the IrcParser
func (p *IrcParser) RemoveIrcHandler(handler string) error {
	handler = strings.ToLower(handler)
	_, has := p.handlers[handler]
	if !has {
		return errHandlerNotRegistered
	}

	delete(p.handlers, handler)
	return nil
}

// parse: Parses an irc protocol string
func (p *IrcParser) Parse(ircproto string) (*ParseResult, error) {
	if len(ircproto) == 0 {
		return nil, errNoProtocolGiven
	}

	splits := strings.Split(ircproto, " ")
	result := createParseResult()
	result.Name = strings.ToLower(splits[0])
	chain, ok := p.handlers[result.Name]
	if !ok {
		return nil, errHandlerNotRegistered
	}
	err := walkProto(chain, splits[1:], result)
	return result, err
}

// walkProto: Walks the protocol tokens and parse tree to fill a ParseResult
func walkProto(chain *fragment, proto []string, result *ParseResult) error {
	frag, i, err := walkHelper(chain, 0, proto, result)
	if err != nil {
		return err
	}
	if frag != nil && frag.optional == nil && i >= len(proto) {
		return errExpectedMoreArguments
	}
	return nil
}

// walkHelper: Recursive function for walkProto
func walkHelper(
	chain *fragment, i int,
	proto []string, result *ParseResult) (*fragment, int, error) {

	var err error = nil
	for chain != nil && i < len(proto) {
		if chain.optional != nil {
			_, i, err = walkHelper(chain.optional, i, proto, result)
			if err != nil {
				break
			}
			chain = chain.next
			continue
		}

		if chain.final {
			var value string
			value, err = handleFinalChain(i, proto)
			if err != nil {
				break
			}
			result.Args[chain.id] = value
			i = len(proto)
		} else {
			if chain.channel {
				result.Channels[chain.id] = strings.Split(proto[i], ",")
			} else if chain.args {
				result.Argv[chain.id] = strings.Split(proto[i], ",")
			}
			result.Args[chain.id] = proto[i]
		}

		chain = chain.next
		i++
	}
	return chain, i, err
}

func handleFinalChain(index int, proto []string) (string, error) {
	var value string
	if strings.HasPrefix(proto[index], ":") {
		value = proto[index][1:] + " " + strings.Join(proto[index+1:], " ")
	} else if index+1 != len(proto) {
		return "", errArgsAfterFinalNoColon
	} else {
		value = proto[index]
	}
	return value, nil
}
