package noise

import (
	n "github.com/waku-org/noise"
)

/*
K1K1:

	->  s
	<-  s
	   ...
	->  e
	<-  e, ee, es
	->  se
*/
var HandshakeK1K1 = n.HandshakePattern{
	Name:                 "K1K1",
	InitiatorPreMessages: []n.MessagePattern{n.MessagePatternS},
	ResponderPreMessages: []n.MessagePattern{n.MessagePatternS},
	Messages: [][]n.MessagePattern{
		{n.MessagePatternE},
		{n.MessagePatternE, n.MessagePatternDHEE, n.MessagePatternDHES},
		{n.MessagePatternDHSE},
	},
}

/*
XK1:

	<-  s
	   ...
	->  e
	<-  e, ee, es
	->  s, se
*/
var HandshakeXK1 = n.HandshakePattern{
	Name:                 "XK1",
	ResponderPreMessages: []n.MessagePattern{n.MessagePatternS},
	Messages: [][]n.MessagePattern{
		{n.MessagePatternE},
		{n.MessagePatternE, n.MessagePatternDHEE, n.MessagePatternDHES},
		{n.MessagePatternS, n.MessagePatternDHSE},
	},
}

/*
XX:

	->  e
	<-  e, ee, s, es
	->  s, se
*/
var HandshakeXX = n.HandshakePattern{
	Name: "XX",
	Messages: [][]n.MessagePattern{
		{n.MessagePatternE},
		{n.MessagePatternE, n.MessagePatternDHEE, n.MessagePatternS, n.MessagePatternDHES},
		{n.MessagePatternS, n.MessagePatternDHSE},
	},
}

/*
XXpsk0:

	->  psk, e
	<-  e, ee, s, es
	->  s, se
*/
var HandshakeXXpsk0 = n.HandshakePattern{
	Name: "XXpsk0",
	Messages: [][]n.MessagePattern{
		{n.MessagePatternPSK, n.MessagePatternE},
		{n.MessagePatternE, n.MessagePatternDHEE, n.MessagePatternS, n.MessagePatternDHES},
		{n.MessagePatternS, n.MessagePatternDHSE},
	},
}
