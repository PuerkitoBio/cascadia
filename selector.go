package cascadia

import (
	"fmt"
	"html"
	"strings"
)

// the Selector type, and functions for creating them

// A Selector is a function which tells whether a node matches or not.
type Selector func(*html.Node) bool

// Compile parses a selector and returns, if successful, a Selector object
// that can be used to match against html.Node objects.
func Compile(sel string) (Selector, error) {
	p := &parser{s: sel}
	compiled, err := p.parseSimpleSelectorSequence() // TODO: more complicated selectors
	if err != nil {
		return nil, err
	}

	if p.i < len(sel) {
		return nil, fmt.Errorf("parsing %q: %d bytes left over", sel, len(sel)-p.i)
	}

	return compiled, nil
}

// MatchAll returns a slice of the nodes that match the selector,
// from n and its children.
func (s Selector) MatchAll(n *html.Node) (result []*html.Node) {
	if s(n) {
		result = append(result, n)
	}

	for _, child := range n.Child {
		result = append(result, s.MatchAll(child)...)
	}

	return
}

// typeSelector returns a Selector that matches elements with a given tag name.
func typeSelector(tag string) Selector {
	tag = toLowerASCII(tag)
	return func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == tag
	}
}

// toLowerASCII returns s with all ASCII capital letters lowercased.
func toLowerASCII(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if c := s[i]; 'A' <= c && c <= 'Z' {
			if b == nil {
				b = make([]byte, len(s))
				copy(b, s)
			}
			b[i] = s[i] + ('a' - 'A')
		}
	}

	if b == nil {
		return s
	}

	return string(b)
}

// attributeSelector returns a Selector that matches elements
// where the attribute named key satisifes the function f.
func attributeSelector(key string, f func(string) bool) Selector {
	key = toLowerASCII(key)
	return func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		for _, a := range n.Attr {
			if a.Key == key && f(a.Val) {
				return true
			}
		}
		return false
	}
}

// attributeExistsSelector returns a Selector that matches elements that have
// an attribute named key.
func attributeExistsSelector(key string) Selector {
	return attributeSelector(key, func(string) bool { return true })
}

// attributeEqualsSelector returns a Selector that matches elements where
// the attribute named key has the value val.
func attributeEqualsSelector(key, val string) Selector {
	return attributeSelector(key,
		func(s string) bool {
			return s == val
		})
}

// attributeIncludesSelector returns a Selector that matches elements where 
// the attribute named key is a whitespace-separated list that includes val.
func attributeIncludesSelector(key, val string) Selector {
	return attributeSelector(key,
		func(s string) bool {
			for s != "" {
				i := strings.IndexAny(s, " \t\r\n\f")
				if i == -1 {
					return s == val
				}
				if s[:i] == val {
					return true
				}
				s = s[i+1:]
			}
			return false
		})
}

// attributeDashmatchSelector returns a Selector that matches elements where
// the attribute named key equals val or starts with val plus a hyphen.
func attributeDashmatchSelector(key, val string) Selector {
	return attributeSelector(key,
		func(s string) bool {
			if s == val {
				return true
			}
			if len(s) <= len(val) {
				return false
			}
			if s[:len(val)] == val && s[len(val)] == '-' {
				return true
			}
			return false
		})
}

// attributePrefixSelector returns a Selector that matches elements where
// the attribute named key starts with val.
func attributePrefixSelector(key, val string) Selector {
	return attributeSelector(key,
		func(s string) bool {
			return strings.HasPrefix(s, val)
		})
}

// attributeSuffixSelector returns a Selector that matches elements where
// the attribute named key ends with val.
func attributeSuffixSelector(key, val string) Selector {
	return attributeSelector(key,
		func(s string) bool {
			return strings.HasSuffix(s, val)
		})
}

// attributeSubstringSelector returns a Selector that matches nodes where
// the attribute named key contains val.
func attributeSubstringSelector(key, val string) Selector {
	return attributeSelector(key,
		func(s string) bool {
			return strings.Contains(s, val)
		})
}

// intersectionSelector returns a selector that matches nodes that match
// both a and b.
func intersectionSelector(a, b Selector) Selector {
	return func(n *html.Node) bool {
		return a(n) && b(n)
	}
}

// negatedSelector returns a selector that matches nodes that do not match a.
func negatedSelector(a Selector) Selector {
	return func(n *html.Node) bool {
		return !a(n)
	}
}

// nthChildSelector returns a selector that implements :nth-child(an+b).
// If last is true, implements :nth-last-child instead.
// If ofType is true, implements :nth-of-type instead.
func nthChildSelector(a, b int, last, ofType bool) Selector {
	return func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}

		parent := n.Parent
		if parent == nil {
			return false
		}

		i := -1
		count := 0
		for _, c := range parent.Child {
			if (c.Type != html.ElementNode) || (ofType && c.Data != n.Data) {
				continue
			}
			count++
			if c == n {
				i = count
				if !last {
					break
				}
			}
		}

		if i == -1 {
			// This shouldn't happen, since n should always be one of its parent's children.
			return false
		}

		if last {
			i = count - i + 1
		}

		i -= b
		if a == 0 {
			return i == 0
		}

		return i%a == 0 && i/a >= 0
	}
}

// onlyChildSelector returns a selector that implements :only-child.
// If ofType is true, it implements :only-of-type instead.
func onlyChildSelector(ofType bool) Selector {
	return func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}

		parent := n.Parent
		if parent == nil {
			return false
		}

		count := 0
		for _, c := range parent.Child {
			if (c.Type != html.ElementNode) || (ofType && c.Data != n.Data) {
				continue
			}
			count++
			if count > 1 {
				return false
			}
		}

		return count == 1
	}
}
