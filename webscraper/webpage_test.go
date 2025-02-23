package webscraper

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/jarcoal/httpmock"
	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/logger"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

var errReader = errors.New("error from reader")

type reader struct{}

func (r *reader) Read(p []byte) (n int, err error) { return 0, errReader }

func TestLoad(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponse("GET", "http://localhost.la/404", 404, "")
	cli.AddMockedResponse("GET", "http://localhost.la/500", 500, "")
	cli.AddMockedResponse("GET", "http://localhost.la/502", 502, "")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetLinks("a")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "404", url.Values{})
	c.Equal(ErrPageNotFound, err)

	err = page.Load(context.Background(), "500", url.Values{})
	c.True(errors.Is(err, ErrInternalServerError))

	err = page.Load(context.Background(), "502", url.Values{})
	c.True(errors.Is(err, ErrInternalServerError))
}

func TestLoadWithHeaders(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponse("GET", "http://localhost.la/404", 404, "")
	cli.AddMockedResponse("GET", "http://localhost.la/500", 500, "")
	cli.AddMockedResponse("GET", "http://localhost.la/502", 502, "")

	header := http.Header{
		"accept-language": {"en-US,en;q=0.9,es;q=0.8,und;q=0.7"},
	}

	fakePage := NewWebPage("http://localhost.fake", cli, logger.New("webscraper"))
	err = fakePage.LoadWithHeaders(context.Background(), "200", header, url.Values{})
	c.Error(ErrPageNotFound, err)

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetLinks("a")
	c.Error(err, ErrPageNotLoaded)

	err = page.LoadWithHeaders(context.Background(), "404", header, url.Values{})
	c.Equal(ErrPageNotFound, err)

	err = page.LoadWithHeaders(context.Background(), "500", header, url.Values{})
	c.True(errors.Is(err, ErrInternalServerError))

	err = page.LoadWithHeaders(context.Background(), "502", header, url.Values{})
	c.True(errors.Is(err, ErrInternalServerError))
}

func TestLogRequestBlocked(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	err = page.LogRequestBlocked(context.Background())
	c.Equal(client.ErrInvalidResponse, err)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)
	c.Equal("http://localhost.la/simple.html", page.GetFullURL())

	err = page.LogRequestBlocked(context.Background())
	c.Nil(err)
}

var cleanNumbersRegexp = regexp.MustCompile(`(\d)`)

func censureNumbers(bin []byte) []byte {
	r := cleanNumbersRegexp.ReplaceAll(bin, []byte("*"))

	return r
}

func TestLoadWithCleaner(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.NoError(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponse("GET", "http://localhost.la/200", 200, "<html><body><p>1 + 1 = 2</p></body></html>")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	page.SetBodyCleanerFunc(censureNumbers)

	err = page.Load(context.Background(), "200", url.Values{})
	c.NoError(err)

	r := page.GetRawText("p")
	c.Equal("* + * = *", r)
}

func TestLoadFromReader(t *testing.T) {
	c := require.New(t)

	page := &WebPage{}
	page.SetDecoder(charmap.Windows1252)

	err := page.loadFromReader(&reader{})
	c.Equal(errReader, err)

	r := strings.NewReader("<div>hi</div>")
	err = page.loadFromReader(r)
	c.Nil(err)

	page.SetBodyCleanerFunc(censureNumbers)
	err = page.loadFromReader(&reader{})
	c.Equal(errReader, err)
}

func TestLoadErrorInRequest(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	u := "http://localhost.la/"

	expectedError := errors.New("test")

	httpmock.RegisterNoResponder(func(r *http.Request) (*http.Response, error) {
		return &http.Response{}, expectedError
	})

	page := NewWebPage(u, cli, logger.New("test"))
	_, err = page.GetLinks("a")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "", url.Values{})
	c.Error(err)
	c.Contains(err.Error(), expectedError.Error())
}

func TestGetText(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetText("body")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)
	c.Equal("http://localhost.la/simple.html", page.GetFullURL())

	txt, err := page.GetText("body")
	c.Nil(err)
	c.Equal(txt, "Hello World!")

	txt, err = page.GetText(".div-with-text")
	c.Nil(err)
	c.Equal("Div Text", txt)

	txt, err = page.GetText(".div-with-span")
	c.Nil(err)
	c.Equal("", txt)
}

func TestGetHTML(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)
	c.Equal("http://localhost.la/simple.html", page.GetFullURL())

	txtExpected := "<!DOCTYPE html><html lang=\"en\"><head>\n  <meta charset=\"utf-8\"/>\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1, user-scalable=yes\"/>\n\n  <title>Hello World!</title>\n</head>\n\n<body>\n\n  Hello World!\n\n  <p>Example</p>\n\n  <div class=\"div-with-text\">Div Text</div>\n  <div class=\"div-with-span\">\n    <span>Span Text</span>\n    <span class=\"span-with-class\">Span With Class Text</span>\n  </div>\n\n  <ul>\n    <li class=\"multiple-line\">\n      Test multiple\n      line\n    </li>\n  </ul>\n  <input id=\"input\" type=\"radio\" value=\"Input Text\" checked=\"checked\"/>\n\n\n\n</body></html>"

	txt, err := page.GetHTML()
	c.Nil(err)
	c.Equal(txtExpected, txt)

	page.doc = nil
	txt, err = page.GetHTML()
	c.EqualError(ErrPageNotLoaded, err.Error())
	c.Empty(txt)
}

func TestGetLinks(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)
	c.Equal("http://localhost.la/table.html", page.GetFullURL())

	links, err := page.GetLinks("#table2 a")
	c.Nil(err)
	c.Contains(links, "/a")
	c.NotContains(links, "/d")
}

func TestContainsNodes(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.ContainsNodes("body p")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	ok, err := page.ContainsNodes("body p")
	c.Nil(err)
	c.True(ok)

	ok, err = page.ContainsNodes("body strong")
	c.Nil(err)
	c.False(ok)
}

func TestContainsText(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	found := page.ContainsText("body", "Hello")
	c.True(found)

	found = page.ContainsText("body p", "Hello")
	c.False(found)

	found = page.ContainsText(".multiple-line", "Test multiple line")
	c.True(found)
}

func TestGetTable(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	table, err := page.GetTable("#table1")
	c.Nil(err)
	c.Equal("1.1", table[0][0])
	c.Equal("1.2", table[0][1])

	table, err = page.GetTable("#table2")
	c.Nil(err)
	c.Equal("", table[0][0])
	c.Equal("", table[1][0])
	c.Equal("", table[2][0])
}

func TestGetRawTable(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	table, err := page.GetRawTable("#table1")
	c.Nil(err)
	c.Equal(table[0][0], "1.1")
	c.Equal(table[0][1], "1.2")

	table, err = page.GetRawTable("#table2")
	c.Nil(err)
	c.Equal(table[0][0], "A")
	c.Equal(table[1][0], "B")
	c.Equal(table[2][0], "C")
}

func TestGetRowTableText(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetRowTableText("#table1", "tr", "td")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	table, err := page.GetRowTableText("#table1", "tr", "td")
	c.Nil(err)
	c.Equal(table[0][0], "1.1")
	c.Equal(table[0][1], "1.2")

	table, err = page.GetRowTableText("#table1", "tr", "th")
	c.Nil(err)
	c.Equal(table[0][0], "Header 1")
	c.Equal(table[0][1], "Header 2")
}

func TestGetHTMLTable(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	table, err := page.GetHTMLTable("#table1")
	c.Nil(err)
	c.Equal("1.1", table[0][0])
	c.Equal("1.2", table[0][1])

	table, err = page.GetHTMLTable("#table2")
	c.Nil(err)
	c.Equal("<a href=\"/a\">A</a>\n    ", table[0][0])
	c.Equal("<a href=\"/b\">B</a>", table[1][0])
	c.Equal("<a href=\"/c\">C</a>", table[2][0])

	table, err = page.GetHTMLTableWithHeaders("#table1")
	c.Nil(err)
	c.Equal("Header 1", table[0][0])
	c.Equal("Header 2", table[0][1])
	c.Equal("1.1", table[1][0])
	c.Equal("1.2", table[1][1])
}

func TestGetTextForEachElement(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/text.html", 200, "samples/text.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTextForEachElement(".test p", -1)
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "text.html", url.Values{})
	c.Nil(err)

	data, err := page.GetTextForEachElement(".test p", -1)
	c.Nil(err)

	labels := []string{"Working", "At", "Example", "Rocks"}
	for index, label := range labels {
		c.Equal(label, data[index])
	}

	data, err = page.GetTextForEachElement(".test p", 3)
	c.Nil(err)

	labels = []string{"Working", "At", "Example"}
	for index, label := range labels {
		c.Equal(label, data[index])
	}

	c.Len(data, 3)
}

func TestGetRawTextForEachElement(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/text.html", 200, "samples/table_should_raw.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	selector := "tr > td:nth-child(2)"

	_, err = page.GetTextForEachElement(selector, -1)
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "text.html", nil)
	c.NoError(err)

	_, err = page.GetTextForEachElement(selector, -1)
	c.NoError(err)

	text, err := page.GetRawTextForEachElement(selector, 4)
	c.NoError(err)
	c.Len(text, 4)

	for _, t := range text {
		c.NotEmpty(t)
	}

	// using normal get text will not work
	text, err = page.GetTextForEachElement(selector, 4)
	c.NoError(err)
	c.Len(text, 4)

	for _, t := range text {
		c.Empty(t)
	}
}

func BenchmarkGetTableSuccess(b *testing.B) {
	c := require.New(b)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	for i := 0; i < b.N; i++ {
		_, err = page.GetTable("#table1")
		c.Nil(err)
	}
}

func TestUnlimiGettList(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/list.html", 200, "samples/list.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "list.html", url.Values{})
	c.Nil(err)

	list, err := page.GetList("#list", 0)
	c.Nil(err)
	c.Equal(len(list), 3)
	c.Equal(list[0], "A")
	c.Equal(list[1], "B")
	c.Equal(list[2], "C")
}

func TestUnlimiGetListRaw(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/list.html", 200, "samples/list.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "list.html", url.Values{})
	c.Nil(err)

	list, err := page.GetListRaw("#list", 0)
	c.Nil(err)
	c.Equal(len(list), 3)
	c.Equal(list[0], "A")
	c.Equal(list[1], "B")
	c.Equal(list[2], "C")
}

func TestLimiGettList(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/list.html", 200, "samples/list.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "list.html", url.Values{})
	c.Nil(err)

	list, err := page.GetList("#list", 2)
	c.Nil(err)
	c.Equal(len(list), 2)
	c.Equal(list[0], "A")
	c.Equal(list[1], "B")
}

func TestLimiGettListRaw(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/list.html", 200, "samples/list.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetTable("#table1")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "list.html", url.Values{})
	c.Nil(err)

	list, err := page.GetListRaw("#list", 2)
	c.Nil(err)
	c.Equal(len(list), 2)
	c.Equal(list[0], "A")
	c.Equal(list[1], "B")
}

func TestGetRawText(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/list.html", 200, "samples/list.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	txt := page.GetRawText("#list")
	c.Equal("", txt)

	err = page.Load(context.Background(), "list.html", url.Values{})
	c.Nil(err)

	txt = page.GetRawText("#list")

	c.Nil(err)
	c.Equal(txt, "A B C")
}

func TestGetAttr(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetText("body")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	txt, err := page.GetAttr("#input", "value")
	c.Nil(err)
	c.Equal(txt, "Input Text")

	txt, err = page.GetAttr("#_notexist", "value")
	c.Equal("", txt)
	c.Equal(ErrElementNotFound, err)
}

func TestCountNodes(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/list.html", 200, "samples/list.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.CountNodes("li")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "list.html", url.Values{})
	c.Nil(err)

	nodesNumber, err := page.CountNodes("li")
	c.Nil(err)
	c.Equal(3, nodesNumber)
}

func TestBuildURL(t *testing.T) {
	c := require.New(t)

	page := NewWebPage("http://host.com", nil, nil)

	newPath, err := page.BuildURL("")
	c.Nil(err)
	c.Equal("http://host.com", newPath)

	newPath, err = page.BuildURL("page.html")
	c.Nil(err)
	c.Equal("http://host.com/page.html", newPath)

	newPath, err = page.BuildURL("page.html?")
	c.Nil(err)
	c.Equal("http://host.com/page.html", newPath)

	newPath, err = page.BuildURL("page.html?query=1")
	c.Nil(err)
	c.Equal("http://host.com/page.html?query=1", newPath)

	// Query ending in question mark
	newPath, err = page.BuildURL("?query=1?")
	c.Nil(err)
	c.Equal("http://host.com?query=1?", newPath)

	// Another url with other host
	newPath, err = page.BuildURL("https://otherhost.com")
	c.Nil(err)
	c.Equal("http://host.com", newPath)

	// Another url with other host and query
	newPath, err = page.BuildURL("https://otherhost.com/?query=1")
	c.Nil(err)
	c.Equal("http://host.com/?query=1", newPath)

	// Invalid URL
	_, err = page.BuildURL("https\\://otherhost.com/?query=1")
	c.NotNil(err)

	urlErr := &url.Error{}

	c.True(errors.As(err, &urlErr))

	errStr := err.Error()
	c.Contains(errStr, "first path segment in URL cannot contain colon")
}

func TestLoadFromFile(t *testing.T) {
	c := require.New(t)

	filePath := "samples/dummy.txt"
	file := `<html><div id="test"> test </div> </html>`

	readFile = func(path string) ([]byte, error) {
		c.Equal(filePath, path)
		return []byte(file), nil
	}

	defer func() { readFile = os.ReadFile }()

	page := WebPage{}
	err := page.LoadFromFile(filePath)
	c.NoError(err)
	c.Equal("test", page.GetRawText("#test"))
}

func TestLoadFromFileError(t *testing.T) {
	c := require.New(t)

	filePath := "samples/dummy.txt"
	expectedError := errors.New("test")

	readFile = func(path string) ([]byte, error) {
		c.Equal(filePath, path)
		return nil, expectedError
	}

	defer func() { readFile = os.ReadFile }()

	page := WebPage{}
	err := page.LoadFromFile(filePath)
	c.Equal(expectedError, err)
}

func TestGetChildrenFiltered(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetChildrenFiltered("div", "span", 1)
	c.Error(ErrPageNotLoaded, err)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	text, err := page.GetChildrenFiltered("div", "span", 1)
	c.Nil(err)
	c.Equal("Span TextSpan With Class Text", text)
}

func TestGetAttrsWithOptions(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetAttrsWithOptions("input", QueryOptions{})
	c.Error(ErrPageNotLoaded, err)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	attrs, err := page.GetAttrsWithOptions("input", QueryOptions{
		Attr:  "value",
		Clean: true,
	})
	c.Nil(err)
	c.Equal(1, len(attrs))
	c.Equal("Input Text", attrs[0])

	attrs, err = page.GetAttrsWithOptions("input", QueryOptions{
		SubSelector: "a",
		Attr:        "nonExistentAttr",
	})
	c.Nil(err)
	c.Equal(0, len(attrs))
}

func TestGetAttrsWithOptionsFiltered(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	attrs, err := page.GetAttrsWithOptions("input", QueryOptions{
		Attr: "value",
		TextFilter: func(t string) bool {
			return false
		},
	})
	c.Nil(err)
	c.Equal(0, len(attrs))
}

func TestGetAttrsWithOptionsSkipAndLimit(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	attrs, err := page.GetAttrsWithOptions("tr td a", QueryOptions{
		Attr:       "href",
		Skip:       1,
		Limit:      1,
		TextFilter: func(t string) bool { return len(t) == 1 },
	})
	c.Nil(err)
	c.Equal(1, len(attrs))
	c.Equal("/b", attrs[0])
}

func TestGetRawBodyNotLoad(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	_, err = page.GetRawBody()

	c.Equal(ErrPageNotLoaded, err)
}

func TestGetRawBody(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	_, err = page.GetRawBody()
	c.Nil(err)
}

func TestGetHTMLRemovingSelectorsNotLoad(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/getHTML.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	childrenSelectors := []string{
		"div",
		"a",
	}

	_, err = page.GetHTMLRemovingSelectors("#divConteudo", childrenSelectors)
	c.Equal(ErrPageNotLoaded, err)
}

func TestGetHTMLRemovingSelectors(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/getHTML.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	err = page.Load(context.Background(), "table.html", url.Values{})
	c.Nil(err)

	childrenSelectors := []string{
		"div",
		"a",
	}

	document, err := page.GetHTMLRemovingSelectors("#divConteudo", childrenSelectors)

	c.Nil(err)
	c.Equal("<p>Working</p> <p>At</p> <p>Example</p> <p>Rocks</p>", document)
}

func TestGetAttrTextForEachElement(t *testing.T) {
	c := require.New(t)
	cli, err := client.New()
	c.NoError(err)

	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/select.html", 200, "samples/select.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetAttrTextForEachElement("option", "value", 0)
	c.Error(ErrPageNotLoaded, err)

	err = page.Load(context.Background(), "select.html", url.Values{})
	c.NoError(err)

	data, err := page.GetAttrTextForEachElement("option", "value", 0)
	c.NoError(err)

	labels := []string{"74603", "535782"}
	for index, label := range labels {
		c.Equal(label, data[index])
	}

	c.Len(data, 2)

	data, err = page.GetAttrTextForEachElement("option", "value", 1)
	c.NoError(err)

	labels = []string{"74603"}
	for index, label := range labels {
		c.Equal(label, data[index])
	}

	c.Len(data, 1)
}

func TestGetDoc(t *testing.T) {
	c := require.New(t)

	doc := &goquery.Document{}

	page := &WebPage{
		doc: doc,
	}

	c.Equal(doc, page.GetDoc())
}

func TestLoadFromString(t *testing.T) {
	c := require.New(t)

	page := &WebPage{}

	err := page.LoadFromString(`<html><body><p id="work">Working</p><p>At</p><p>Example</p><p>Rocks</p></html></body>`)
	c.NoError(err)

	txt, err := page.GetText("#work")
	c.NoError(err)
	c.Equal("Working", txt)
}

func TestGetElementsByProperties(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("div", map[string]string{"class": "div-with-text"})
	c.Equal(1, len(elements))
}

func TestGetElementFromParentDepth(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("div", map[string]string{"class": "div-with-text"})
	c.Equal(1, len(elements))

	parent := page.GetParentElementWithDepth(elements[0], 1)
	c.Contains(parent.Text(), "Example")
}

func TestGetElementFromNextSiblingDepth(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("div", map[string]string{"class": "div-with-text"})
	c.Equal(1, len(elements))

	sibling := page.GetNextSiblingElementWithDepth(elements[0], 1)
	c.Equal(cleanSpaces(sibling.Text()), "Span Text Span With Class Text")
}

func TestGetElementFromPrevSiblingDepth(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("div", map[string]string{"class": "div-with-span"})
	c.Equal(1, len(elements))

	sibling := page.GetPrevSiblingElementWithDepth(elements[0], 1)
	c.Equal(cleanSpaces(sibling.Text()), "Div Text")
}

func TestGetElementByPropertiesFromSelection(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("div", map[string]string{"class": "div-with-span"})
	c.Equal(1, len(elements))

	foundElement := page.GetElementByAttributesFromSelection(elements[0], "span", map[string]string{"class": "span-with-class"})
	c.Equal(cleanSpaces(page.GetTextFromSelection(foundElement)), "Span With Class Text")
}

func TestGetTablesFromSelection(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("body", map[string]string{"class": "test"})
	c.Equal(1, len(elements))

	tables := page.GetTablesFromSelection(elements[0])
	c.Equal(len(tables), 3)
}

func TestGetTableDivDataFromSelection(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("body", map[string]string{"class": "test"})
	c.Equal(1, len(elements))

	tables := page.GetTablesFromSelection(elements[0])
	c.Equal(len(tables), 3)

	tableData := page.GetTableDivDataFromSelection(tables[2])
	c.Equal(1, len(tableData))
}

func TestGetTextFromSelectionTag(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/simple.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	_, err = page.GetText("body")
	c.Error(err, ErrPageNotLoaded)

	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)
	c.Equal("http://localhost.la/simple.html", page.GetFullURL())

	body := page.GetElementsBySelector("body")
	c.IsType([]*goquery.Selection{}, body)

	txt := page.GetTextFromSelectionTag(body[0], "div")
	c.Equal("Div Text", txt)
}

func TestGetTableDataFromSelection(t *testing.T) {
	c := require.New(t)

	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/simple.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))
	err = page.Load(context.Background(), "simple.html", url.Values{})
	c.Nil(err)

	elements := page.GetElementsByAttributes("body", map[string]string{"class": "test"})
	c.Equal(1, len(elements))

	tables := page.GetTablesFromSelection(elements[0])
	c.Equal(len(tables), 3)

	tableData := page.GetTableDataFromSelection(tables[2])
	c.Equal(1, len(tableData))
}

func BenchmarkGetRawBody(b *testing.B) {
	c := require.New(b)
	cli, err := client.New()
	c.Nil(err)

	// Activate mock
	cli.ActivateMock()
	defer cli.DeactivateMock()

	cli.AddMockedResponseFromFile("GET", "http://localhost.la/table.html", 200, "samples/table.html")

	page := NewWebPage("http://localhost.la", cli, logger.New("webscraper"))

	err = page.Load(context.Background(), "table.html", url.Values{})
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := page.GetRawBody()
		if err != nil {
			b.Fatal(err)
		}
	}
}
