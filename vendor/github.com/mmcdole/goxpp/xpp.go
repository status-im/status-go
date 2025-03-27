package xpp

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
)

type XMLEventType int
type CharsetReader func(charset string, input io.Reader) (io.Reader, error)

const xmlNSURI = "http://www.w3.org/XML/1998/namespace"

const (
	StartDocument XMLEventType = iota
	EndDocument
	StartTag
	EndTag
	Text
	Comment
	ProcessingInstruction
	Directive
	IgnorableWhitespace // TODO: ?
	// TODO: CDSECT ?
)

type urlStack []*url.URL

func (s *urlStack) push(u *url.URL) {
	*s = append([]*url.URL{u}, *s...)
}

func (s *urlStack) pop() *url.URL {
	if s == nil || len(*s) == 0 {
		return nil
	}
	var top *url.URL
	top, *s = (*s)[0], (*s)[1:]
	return top
}

func (s *urlStack) Top() *url.URL {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return (*s)[0]
}

type XMLPullParser struct {
	// Document State
	Spaces      map[string]string
	SpacesStack []map[string]string
	BaseStack   urlStack

	// Token State
	Depth int
	Event XMLEventType
	Attrs []xml.Attr
	Name  string
	Space string
	Text  string

	decoder *xml.Decoder
	token   interface{}
}

func NewXMLPullParser(r io.Reader, strict bool, cr CharsetReader) *XMLPullParser {
	d := xml.NewDecoder(r)
	d.Strict = strict
	d.CharsetReader = cr
	return &XMLPullParser{
		decoder: d,
		Event:   StartDocument,
		Depth:   0,
		Spaces:  map[string]string{},
	}
}

func (p *XMLPullParser) NextTag() (event XMLEventType, err error) {
	t, err := p.Next()
	if err != nil {
		return event, err
	}

	for t == Text && p.IsWhitespace() {
		t, err = p.Next()
		if err != nil {
			return event, err
		}
	}

	if t != StartTag && t != EndTag {
		return event, fmt.Errorf("Expected StartTag or EndTag but got %s at offset: %d", p.EventName(t), p.decoder.InputOffset())
	}

	return t, nil
}

func (p *XMLPullParser) Next() (event XMLEventType, err error) {
	for {
		event, err = p.NextToken()
		if err != nil {
			return event, err
		}

		// Return immediately after encountering a StartTag
		// EndTag, Text, EndDocument
		if event == StartTag ||
			event == EndTag ||
			event == EndDocument ||
			event == Text {
			return event, nil
		}

		// Skip Comment/Directive and ProcessingInstruction
		if event == Comment ||
			event == Directive ||
			event == ProcessingInstruction {
			continue
		}
	}
	return event, nil
}

func (p *XMLPullParser) NextToken() (event XMLEventType, err error) {
	// Clear any state held for the previous token
	p.resetTokenState()

	token, err := p.decoder.Token()
	if err != nil {
		if err == io.EOF {
			// XML decoder returns the EOF as an error
			// but we want to return it as a valid
			// EndDocument token instead
			p.token = nil
			p.Event = EndDocument
			return p.Event, nil
		}
		return event, err
	}

	p.token = xml.CopyToken(token)
	p.processToken(p.token)
	p.Event = p.EventType(p.token)

	return p.Event, nil
}

func (p *XMLPullParser) NextText() (string, error) {
	if p.Event != StartTag {
		return "", errors.New("Parser must be on StartTag to get NextText()")
	}

	t, err := p.Next()
	if err != nil {
		return "", err
	}

	if t != EndTag && t != Text {
		return "", errors.New("Parser must be on EndTag or Text to read text")
	}

	var result string
	for t == Text {
		result = result + p.Text
		t, err = p.Next()
		if err != nil {
			return "", err
		}

		if t != EndTag && t != Text {
			errstr := fmt.Sprintf("Event Text must be immediately followed by EndTag or Text but got %s", p.EventName(t))
			return "", errors.New(errstr)
		}
	}

	return result, nil
}

func (p *XMLPullParser) Skip() error {
	for {
		tok, err := p.NextToken()
		if err != nil {
			return err
		}
		if tok == StartTag {
			if err := p.Skip(); err != nil {
				return err
			}
		} else if tok == EndTag {
			return nil
		}
	}
}

func (p *XMLPullParser) Attribute(name string) string {
	for _, attr := range p.Attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func (p *XMLPullParser) Expect(event XMLEventType, name string) (err error) {
	return p.ExpectAll(event, "*", name)
}

func (p *XMLPullParser) ExpectAll(event XMLEventType, space string, name string) (err error) {
	if !(p.Event == event && (strings.ToLower(p.Space) == strings.ToLower(space) || space == "*") && (strings.ToLower(p.Name) == strings.ToLower(name) || name == "*")) {
		err = fmt.Errorf("Expected Space:%s Name:%s Event:%s but got Space:%s Name:%s Event:%s at offset: %d", space, name, p.EventName(event), p.Space, p.Name, p.EventName(p.Event), p.decoder.InputOffset())
	}
	return
}

func (p *XMLPullParser) DecodeElement(v interface{}) error {
	if p.Event != StartTag {
		return errors.New("DecodeElement can only be called from a StartTag event")
	}

	//tok := &p.token

	startToken := p.token.(xml.StartElement)

	// Consumes all tokens until the matching end token.
	err := p.decoder.DecodeElement(v, &startToken)
	if err != nil {
		return err
	}

	name := p.Name

	// Need to set the "current" token name/event
	// to the previous StartTag event's name
	p.resetTokenState()
	p.Event = EndTag
	p.Depth--
	p.Name = name
	p.token = nil

	// if the token we decoded had an xml:base attribute, we need to pop it
	// from the stack
	// Note: this means it is up to the caller of DecodeElement to save the curent xml:base
	// before calling DecodeElement if it needs to resolve relative URLs in `v`
	for _, attr := range startToken.Attr {
		if attr.Name.Space == xmlNSURI && attr.Name.Local == "base" {
			p.popBase()
			break
		}
	}
	return nil
}

func (p *XMLPullParser) IsWhitespace() bool {
	return strings.TrimSpace(p.Text) == ""
}

func (p *XMLPullParser) EventName(e XMLEventType) (name string) {
	switch e {
	case StartTag:
		name = "StartTag"
	case EndTag:
		name = "EndTag"
	case StartDocument:
		name = "StartDocument"
	case EndDocument:
		name = "EndDocument"
	case ProcessingInstruction:
		name = "ProcessingInstruction"
	case Directive:
		name = "Directive"
	case Comment:
		name = "Comment"
	case Text:
		name = "Text"
	case IgnorableWhitespace:
		name = "IgnorableWhitespace"
	}
	return
}

func (p *XMLPullParser) EventType(t xml.Token) (event XMLEventType) {
	switch t.(type) {
	case xml.StartElement:
		event = StartTag
	case xml.EndElement:
		event = EndTag
	case xml.CharData:
		event = Text
	case xml.Comment:
		event = Comment
	case xml.ProcInst:
		event = ProcessingInstruction
	case xml.Directive:
		event = Directive
	}
	return
}

// resolve the given string as a URL relative to current xml:base
func (p *XMLPullParser) XmlBaseResolveUrl(u string) (*url.URL, error) {
	curr := p.BaseStack.Top()
	if curr == nil {
		return nil, nil
	}

	relURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	if curr.Path != "" && u != "" && curr.Path[len(curr.Path)-1] != '/' {
		// There's no reason someone would use a path in xml:base if they
		// didn't mean for it to be a directory
		curr.Path = curr.Path + "/"
	}
	absURL := curr.ResolveReference(relURL)
	return absURL, nil
}

func (p *XMLPullParser) processToken(t xml.Token) {
	switch tt := t.(type) {
	case xml.StartElement:
		p.processStartToken(tt)
	case xml.EndElement:
		p.processEndToken(tt)
	case xml.CharData:
		p.processCharDataToken(tt)
	case xml.Comment:
		p.processCommentToken(tt)
	case xml.ProcInst:
		p.processProcInstToken(tt)
	case xml.Directive:
		p.processDirectiveToken(tt)
	}
}

func (p *XMLPullParser) processStartToken(t xml.StartElement) {
	p.Depth++
	p.Attrs = t.Attr
	p.Name = t.Name.Local
	p.Space = t.Name.Space
	p.trackNamespaces(t)
	p.pushBase()
}

func (p *XMLPullParser) processEndToken(t xml.EndElement) {
	p.Depth--
	p.SpacesStack = p.SpacesStack[:len(p.SpacesStack)-1]
	if len(p.SpacesStack) == 0 {
		p.Spaces = map[string]string{}
	} else {
		p.Spaces = p.SpacesStack[len(p.SpacesStack)-1]
	}
	p.Name = t.Name.Local
	p.popBase()
}

func (p *XMLPullParser) processCharDataToken(t xml.CharData) {
	p.Text = string([]byte(t))
}

func (p *XMLPullParser) processCommentToken(t xml.Comment) {
	p.Text = string([]byte(t))
}

func (p *XMLPullParser) processProcInstToken(t xml.ProcInst) {
	p.Text = fmt.Sprintf("%s %s", t.Target, string(t.Inst))
}

func (p *XMLPullParser) processDirectiveToken(t xml.Directive) {
	p.Text = string([]byte(t))
}

func (p *XMLPullParser) resetTokenState() {
	p.Attrs = nil
	p.Name = ""
	p.Space = ""
	p.Text = ""
}

func (p *XMLPullParser) trackNamespaces(t xml.StartElement) {
	newSpace := map[string]string{}
	for k, v := range p.Spaces {
		newSpace[k] = v
	}
	for _, attr := range t.Attr {
		if attr.Name.Space == "xmlns" {
			space := strings.TrimSpace(attr.Value)
			spacePrefix := strings.TrimSpace(strings.ToLower(attr.Name.Local))
			newSpace[space] = spacePrefix
		} else if attr.Name.Local == "xmlns" {
			space := strings.TrimSpace(attr.Value)
			newSpace[space] = ""
		}
	}
	p.Spaces = newSpace
	p.SpacesStack = append(p.SpacesStack, newSpace)
}

// returns the popped base URL
func (p *XMLPullParser) popBase() string {
	url := p.BaseStack.pop()
	if url != nil {
		return url.String()
	}
	return ""
}

// Searches current attributes for xml:base and updates the urlStack
func (p *XMLPullParser) pushBase() error {
	var base string
	// search list of attrs for "xml:base"
	for _, attr := range p.Attrs {
		if attr.Name.Local == "base" && attr.Name.Space == xmlNSURI {
			base = attr.Value
			break
		}
	}
	if base == "" {
		// no base attribute found
		return nil
	}

	newURL, err := url.Parse(base)
	if err != nil {
		return err
	}

	topURL := p.BaseStack.Top()
	if topURL != nil {
		newURL = topURL.ResolveReference(newURL)
	}
	p.BaseStack.push(newURL)
	return nil
}
