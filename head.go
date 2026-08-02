package inertia

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	headOpeningTagPattern = regexp.MustCompile(`^<([a-zA-Z][^\s/>]*)`)
)

// ServerHead is a validated head element collection consumed by Inertia's client serverHead option.
type ServerHead struct {
	elements []string
}

// HeadElement is a safely generated server-provided head element.
type HeadElement struct {
	tag        string
	attributes map[string]string
	content    string
	void       bool
	raw        template.HTML
	rawSet     bool
	err        error
}

// HeadTitle creates an escaped title element.
func HeadTitle(title string) HeadElement {
	return HeadElement{
		tag:        "title",
		attributes: map[string]string{"data-inertia": "title"},
		content:    title,
	}
}

// HeadMeta creates an escaped named meta element with a stable data-inertia key.
func HeadMeta(name string, content string) HeadElement {
	element := HeadElement{}
	if strings.TrimSpace(name) == "" {
		element.err = fmt.Errorf("%w: meta name must not be empty", ErrInvalidHeadElement)
		return element
	}
	return HeadElement{
		tag: "meta",
		attributes: map[string]string{
			"content":      content,
			"data-inertia": "meta:name:" + name,
			"name":         name,
		},
		void: true,
	}
}

// HeadMetaProperty creates an escaped property meta element with a stable data-inertia key.
func HeadMetaProperty(property string, content string) HeadElement {
	element := HeadElement{}
	if strings.TrimSpace(property) == "" {
		element.err = fmt.Errorf("%w: meta property must not be empty", ErrInvalidHeadElement)
		return element
	}
	return HeadElement{
		tag: "meta",
		attributes: map[string]string{
			"content":      content,
			"data-inertia": "meta:property:" + property,
			"property":     property,
		},
		void: true,
	}
}

// HeadLink creates an escaped link element with a stable data-inertia key.
func HeadLink(rel string, href string) HeadElement {
	element := HeadElement{}
	if strings.TrimSpace(rel) == "" || strings.TrimSpace(href) == "" {
		element.err = fmt.Errorf("%w: link rel and href must not be empty", ErrInvalidHeadElement)
		return element
	}
	return HeadElement{
		tag: "link",
		attributes: map[string]string{
			"data-inertia": "link:" + rel + ":" + href,
			"href":         href,
			"rel":          rel,
		},
		void: true,
	}
}

// HeadScript creates an escaped external script element with a stable data-inertia key.
func HeadScript(src string) HeadElement {
	element := HeadElement{}
	if strings.TrimSpace(src) == "" {
		element.err = fmt.Errorf("%w: script src must not be empty", ErrInvalidHeadElement)
		return element
	}
	return HeadElement{
		tag: "script",
		attributes: map[string]string{
			"data-inertia": "script:" + src,
			"src":          src,
		},
	}
}

// RawHeadElement creates an explicitly trusted raw head element.
// Prefer the structured constructors for values containing user input.
func RawHeadElement(value template.HTML) HeadElement {
	return HeadElement{raw: value, rawSet: true}
}

// Attr returns a copy of element with an escaped HTML attribute.
func (e HeadElement) Attr(name string, value string) HeadElement {
	if e.err != nil {
		return e
	}
	if e.rawSet {
		e.err = fmt.Errorf("%w: attributes cannot be added to a raw element", ErrInvalidHeadElement)
		return e
	}
	if !validHeadAttributeName(name) || unsafeHeadAttributeName(name) {
		e.err = fmt.Errorf("%w: invalid attribute name %q", ErrInvalidHeadElement, name)
		return e
	}
	name = strings.ToLower(name)
	if name == "data-inertia" && strings.TrimSpace(value) == "" {
		e.err = fmt.Errorf("%w: data-inertia key must not be empty", ErrInvalidHeadElement)
		return e
	}
	attributes := make(map[string]string, len(e.attributes)+1)
	for key, current := range e.attributes {
		attributes[key] = current
	}
	attributes[name] = value
	e.attributes = attributes
	return e
}

// Key returns a copy of element with a stable data-inertia key.
func (e HeadElement) Key(key string) HeadElement {
	return e.Attr("data-inertia", key)
}

// NewServerHead validates and renders structured elements into the client wire format.
func NewServerHead(elements ...HeadElement) (ServerHead, error) {
	head := ServerHead{elements: make([]string, 0, len(elements))}
	for index, element := range elements {
		rendered, err := element.render()
		if err != nil {
			return ServerHead{}, err
		}
		rendered, err = ensureHeadElementKey(strings.TrimSpace(rendered), index)
		if err != nil {
			return ServerHead{}, err
		}
		head.elements = append(head.elements, rendered)
	}
	return head, nil
}

func ensureHeadElementKey(element string, index int) (string, error) {
	if element == "" {
		return "", fmt.Errorf("%w: element must not be empty", ErrInvalidHeadElement)
	}
	match := headOpeningTagPattern.FindStringSubmatchIndex(element)
	if match == nil {
		return "", fmt.Errorf("%w: element must start with an HTML tag", ErrInvalidHeadElement)
	}
	insertAt := match[3]
	if _, found := openingTagAttribute(element, insertAt, "data-inertia"); found {
		return element, nil
	}
	return element[:insertAt] + ` data-inertia="server-head-` + strconv.Itoa(index) + `"` + element[insertAt:], nil
}

func openingTagAttribute(element string, start int, target string) (string, bool) {
	for index := start; index < len(element); {
		index = skipHTMLSpace(element, index)
		if index >= len(element) || element[index] == '>' {
			return "", false
		}
		if element[index] == '/' {
			index++
			continue
		}

		nameStart := index
		for index < len(element) && !isHTMLSpace(element[index]) && element[index] != '=' && element[index] != '>' && element[index] != '/' {
			index++
		}
		if nameStart == index {
			index++
			continue
		}
		name := element[nameStart:index]
		index = skipHTMLSpace(element, index)

		value := ""
		if index < len(element) && element[index] == '=' {
			index = skipHTMLSpace(element, index+1)
			if index < len(element) && (element[index] == '\'' || element[index] == '"') {
				quote := element[index]
				index++
				valueStart := index
				for index < len(element) && element[index] != quote {
					index++
				}
				value = element[valueStart:index]
				if index < len(element) {
					index++
				}
			} else {
				valueStart := index
				for index < len(element) && !isHTMLSpace(element[index]) && element[index] != '>' {
					index++
				}
				value = element[valueStart:index]
			}
		}

		if strings.EqualFold(name, target) {
			return value, true
		}
	}
	return "", false
}

func skipHTMLSpace(value string, index int) int {
	for index < len(value) && isHTMLSpace(value[index]) {
		index++
	}
	return index
}

func isHTMLSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r' || value == '\f'
}

// Elements returns a copy of the raw HTML strings sent to the Inertia client.
func (h ServerHead) Elements() []string {
	elements := make([]string, len(h.elements))
	copy(elements, h.elements)
	return elements
}

func (h ServerHead) clone() ServerHead {
	return ServerHead{elements: h.Elements()}
}

// MarshalJSON serializes h as the string array expected by the Inertia client.
func (h ServerHead) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Elements())
}

// Props returns the default "head" page prop for h.
func (h ServerHead) Props() Props {
	return Props{"head": h.Elements()}
}

// NamedProps returns a custom top-level page prop for h.
func (h ServerHead) NamedProps(name string) (Props, error) {
	if !validServerHeadProp(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidServerHeadProp, name)
	}
	return Props{name: h.Elements()}, nil
}

// HTML returns the elements for the initial root template fallback.
func (h ServerHead) HTML() template.HTML {
	return template.HTML(strings.Join(h.elements, "\n"))
}

func (e HeadElement) render() (string, error) {
	if e.err != nil {
		return "", e.err
	}
	if e.rawSet {
		if e.raw == "" {
			return "", fmt.Errorf("%w: raw element must not be empty", ErrInvalidHeadElement)
		}
		return string(e.raw), nil
	}
	if e.tag == "" {
		return "", fmt.Errorf("%w: missing tag", ErrInvalidHeadElement)
	}
	keys := make([]string, 0, len(e.attributes))
	for key := range e.attributes {
		if !validHeadAttributeName(key) || unsafeHeadAttributeName(key) {
			return "", fmt.Errorf("%w: invalid attribute name %q", ErrInvalidHeadElement, key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteByte('<')
	builder.WriteString(e.tag)
	for _, key := range keys {
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteString(`="`)
		builder.WriteString(html.EscapeString(e.attributes[key]))
		builder.WriteByte('"')
	}
	builder.WriteByte('>')
	if e.void {
		return builder.String(), nil
	}
	builder.WriteString(html.EscapeString(e.content))
	builder.WriteString("</")
	builder.WriteString(e.tag)
	builder.WriteByte('>')
	return builder.String(), nil
}

func unsafeHeadAttributeName(value string) bool {
	normalized := strings.ToLower(value)
	return strings.HasPrefix(normalized, "on") || normalized == "srcdoc"
}

func validHeadAttributeName(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character == '_' || character == ':' {
			continue
		}
		if index > 0 && (character >= '0' && character <= '9' || character == '-' || character == '.') {
			continue
		}
		return false
	}
	return true
}

func validServerHeadProp(value string) bool {
	if value == "" || value == "errors" || value == "flash" {
		return false
	}
	for index, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character == '_' {
			continue
		}
		if index > 0 && (character >= '0' && character <= '9' || character == '-') {
			continue
		}
		return false
	}
	return true
}
