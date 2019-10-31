package crawl

import (
	"testing"

	"golang.org/x/net/html"
)

type linksTest struct {
	valid   string
	invalid string
	errMsg  string
}

// TestSanitiseFail tests special cases with special characters found in URL, that should bring URL.Parse() to fail
func TestSanitiseFail(t *testing.T) {
	tests := []linksTest{
		{"https://example.com/resource/is-here", "https://example.com/\r\n",
			"sanitise() should fail on calling url.Parse() with error from parse() : contains ASCII control characters."},
		{"https://example.com/resource/is-here", "https://example.com/%",
			"sanitise() should fail on calling url.Parse() with error from unescape()."},
	}

	// Should fail on url.Parse(origin)
	for _, test := range tests {
		_, serr := sanitise(test.invalid, test.valid)
		if serr == nil {
			t.Errorf("Test on origin parameter : %s - value '%s'", test.errMsg, test.invalid)
		}
	}

	// Should fail on url.Parse(link)
	for _, test := range tests {
		_, serr := sanitise(test.valid, test.invalid)
		if serr == nil {
			t.Errorf("Test on origin parameter : %s - value '%s'", test.errMsg, test.invalid)
		}
	}
}

// TestExtractLink tests special cases when the token arguments has special values or does not contain a link
func TestExtractLink(t *testing.T) {
	testAttribute := html.Attribute{
		Namespace: "",
		Key:       "",
		Val:       "",
	}
	testToken := html.Token{
		Type:     0,
		DataAtom: 0,
		Data:     "",
		Attr:     nil,
	}

	errMsgF := "extractLink() should return an empty string when %s."

	// Should return ""
	// nil slice on token.Attr
	if extractLink("", testToken) != "" {
		t.Errorf(errMsgF, "token.Attr is nil")
	}

	// Should return ""
	// valid token.Attr but no href
	testToken.Attr = []html.Attribute{testAttribute}
	if extractLink("", testToken) != "" {
		t.Errorf(errMsgF, "no href was found")
	}

	// Should return ""
	// valid token.Attr, contains href, but call to sanitise() fails
	testAttribute.Key = "href"
	testAttribute.Val = "%"
	testToken.Attr = []html.Attribute{testAttribute}

	if extractLink("", testToken) != "" {
		t.Errorf(errMsgF, "token.Attr.Val is not valid for sanitise()")
	}
}
