// Package webscraper contains functions to create and manage webscrapers fetchers
package webscraper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/rdaniel1105/gob-vendor/client"
	"golang.org/x/text/encoding/charmap"
)

var (
	// ErrPageNotLoaded error when Load method was not called or it failed
	ErrPageNotLoaded = errors.New("webpage is not loaded")
	// ErrPageNotFound error when the webserver responds with a 404 status
	ErrPageNotFound = errors.New("web page not found")
	// ErrInternalServerError error when the webserver responds with a status >= 500
	ErrInternalServerError = errors.New("web page respond with internal server error")
	// ErrElementNotFound error for element not found
	ErrElementNotFound = errors.New("element not found")

	readFile = os.ReadFile
)

type (
	forEachFunc     func(i int, s *goquery.Selection) bool
	textFilterFunc  func(text string) bool
	appendFunc      func(value string) bool
	bodyCleanerFunc func(body []byte) []byte
)

// WebPage It represents the page to parse
type WebPage struct {
	baseURL     string
	fullURL     string
	doc         *goquery.Document
	client      *client.Client
	logger      *slog.Logger
	decoder     *charmap.Charmap
	bodyCleaner bodyCleanerFunc
	response    *http.Response
}

// QueryOptions represents the configurable options for webpage queries
type QueryOptions struct {
	SubSelector string
	Attr        string
	Clean       bool
	Limit       int
	Skip        int
	TextFilter  textFilterFunc
}

// NewWebPage creates a new web page
func NewWebPage(baseURL string, client *client.Client, logger *slog.Logger) WebPage {
	return WebPage{
		baseURL,
		baseURL,
		nil,
		client,
		logger,
		nil,
		nil,
		nil,
	}
}

// SetDecoder to use in LoadFromReader
func (page *WebPage) SetDecoder(decoder *charmap.Charmap) {
	page.decoder = decoder
}

// LoadWithHeaders fetch web page contents form a path with headers
func (page *WebPage) LoadWithHeaders(ctx context.Context, pathURL string, headers http.Header, params url.Values) error {
	u, err := page.BuildURL(pathURL)
	if err != nil {
		return err
	}

	page.fullURL = u

	response, err := page.client.Get(ctx, u, headers, params)
	if err != nil {
		return err
	}

	page.response = response

	return page.LoadFromResponse(ctx, response)
}

// Load fetch web page contents form a path
func (page *WebPage) Load(ctx context.Context, pathURL string, params url.Values) error {
	u, err := page.BuildURL(pathURL)
	if err != nil {
		return err
	}

	page.fullURL = u

	response, err := page.client.Get(ctx, u, nil, params)
	if err != nil {
		return err
	}

	page.response = response

	return page.LoadFromResponse(ctx, response)
}

// LoadFromResponse load web page contents form a request response
func (page *WebPage) LoadFromResponse(ctx context.Context, response *http.Response) error {
	defer page.CloseOrLogWithContext(ctx, response.Body)

	err := page.loadFromReader(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode == http.StatusNotFound {
		return ErrPageNotFound
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w %d", ErrInternalServerError, response.StatusCode)
	}

	page.response = response

	return nil
}

func (page *WebPage) CloseOrLogWithContext(ctx context.Context, reader io.ReadCloser) {
	if err := reader.Close(); err != nil {
		log.Println("error closing reader", err)
	}
}

// LoadFromFile load web page contents from a filepath for testing purpose
func (page *WebPage) LoadFromFile(path string) error {
	bs, err := readFile(path) // #nosec
	if err != nil {
		return err
	}

	return page.loadFromReader(bytes.NewBuffer(bs))
}

// LoadFromString load web page contents from a html string for testing purpose
func (page *WebPage) LoadFromString(html string) error {
	return page.loadFromReader(bytes.NewBuffer([]byte(html)))
}

// loadFromReader load web page contents from a reader, please do not export this method
func (page *WebPage) loadFromReader(reader io.Reader) error {
	if page.decoder != nil {
		reader = page.decoder.NewDecoder().Reader(reader)
	}

	if page.bodyCleaner != nil {
		return page.loadWithCleaner(reader)
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	page.doc = doc

	return nil
}

func (page *WebPage) loadWithCleaner(reader io.Reader) error {
	d, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	d = page.bodyCleaner(d)

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(d))
	if err != nil {
		return err
	}

	page.doc = doc

	return nil
}

// SetBodyCleanerFunc allows configuring a function to clean the original HTML,
// please use it as the last resource.
func (page *WebPage) SetBodyCleanerFunc(cleaner bodyCleanerFunc) {
	page.bodyCleaner = cleaner
}

// GetAttrs extracts the given attribute contained in the selected nodes of the page
func (page *WebPage) GetAttrs(selector, attr string) ([]string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return []string{}, err
	}

	return page.doc.Find(selector).Map(func(index int, selection *goquery.Selection) string {
		url, _ := selection.Attr(attr)
		return url
	}), nil
}

// GetLinks extracts the href contained in the selected nodes of the page
func (page *WebPage) GetLinks(selector string) ([]string, error) {
	return page.GetAttrs(selector, "href")
}

// ContainsNodes checks whether the page contains at least one node or not
func (page *WebPage) ContainsNodes(selector string) (bool, error) {
	if err := page.checkIfLoaded(); err != nil {
		return false, err
	}

	return len(page.doc.Find(selector).First().Nodes) != 0, nil
}

// ContainsText checks whether a selected node contains the given text
func (page *WebPage) ContainsText(selector string, text string) bool {
	return strings.Contains(cleanSpaces(page.doc.Find(selector).First().Text()), text)
}

// GetText extracts text form a selected node
func (page *WebPage) GetText(selector string) (string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return "", err
	}

	return cleanText(page.doc.Find(selector).First()), nil
}

// CountNodes returns the number of nodes that matches the given selector
func (page *WebPage) CountNodes(selector string) (int, error) {
	if err := page.checkIfLoaded(); err != nil {
		return 0, err
	}

	return page.doc.Find(selector).Length(), nil
}

// GetRawText return all text
func (page *WebPage) GetRawText(selector string) string {
	if err := page.checkIfLoaded(); err != nil {
		return ""
	}

	txt := page.doc.Find(selector).First().Text()

	return cleanSpaces(txt)
}

// GetTable extracts table cells information form a selected node
func (page *WebPage) GetTable(selector string) ([][]string, error) {
	return page.getTableText(selector, true)
}

// GetRawTable extracts raw table cells information form a selected node
func (page *WebPage) GetRawTable(selector string) ([][]string, error) {
	return page.getTableText(selector, false)
}

// GetHTMLTable extracts HTML table cells information form a selected node
func (page *WebPage) GetHTMLTable(selector string) ([][]string, error) {
	return page.getHTMLTable(selector, false)
}

// GetHTMLTableWithHeaders extracts HTML table cells information form a selected node
// It is also able to obtain the values that exist in the table header
func (page *WebPage) GetHTMLTableWithHeaders(selector string) ([][]string, error) {
	return page.getHTMLTable(selector, true)
}

// GetChildrenFiltered gets children filtered with selector
func (page *WebPage) GetChildrenFiltered(parentSelector, childSelector string, index int) (string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return "", err
	}

	return page.doc.Find(parentSelector).Eq(index).ChildrenFiltered(childSelector).Text(), nil
}

func (page *WebPage) getHTMLTable(selector string, head bool) ([][]string, error) {
	var out [][]string

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	rowSelector := "td"
	if head {
		rowSelector += ", th"
	}

	page.doc.Find(selector).Find("tr").Each(func(i int, row *goquery.Selection) {
		info := row.Find(rowSelector).Map(func(index int, selection *goquery.Selection) string {
			data, err := selection.Html()
			if err != nil {
				return ""
			}
			return data
		})

		if len(info) > 0 {
			out = append(out, info)
		}
	})

	return out, nil
}

func (page *WebPage) getTableText(selector string, cleanChildren bool) ([][]string, error) {
	var out [][]string

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).Find("tr").Each(func(i int, row *goquery.Selection) {
		info := row.Find("td").Map(func(index int, selection *goquery.Selection) string {
			if cleanChildren {
				return cleanText(selection)
			}

			return cleanSpaces(selection.Text())
		})

		if len(info) > 0 {
			out = append(out, info)
		}
	})

	return out, nil
}

// GetRowTableText extracts the  information form a selected table with subselector
func (page *WebPage) GetRowTableText(selector string, subSelector string, rowSelector string) ([][]string, error) {
	var out [][]string

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).Find(subSelector).Each(func(i int, row *goquery.Selection) {
		info := row.Find(rowSelector).Map(func(index int, selection *goquery.Selection) string {
			return cleanText(selection)
		})

		if len(info) > 0 {
			out = append(out, info)
		}
	})

	return out, nil
}

// GetList extracts list information form a selected node
func (page *WebPage) GetList(selector string, limit int) ([]string, error) {
	var (
		out     []string
		unLimit = limit <= 0
	)

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).First().Find("li").EachWithBreak(func(i int, row *goquery.Selection) bool {
		if !unLimit && i >= limit {
			return false
		}
		out = append(out, cleanText(row))
		return true
	})

	return out, nil
}

// GetListRaw extracts list information form a selected node without clean it
func (page *WebPage) GetListRaw(selector string, limit int) ([]string, error) {
	var (
		out     []string
		unLimit = limit <= 0
	)

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).First().Find("li").EachWithBreak(func(i int, row *goquery.Selection) bool {
		if !unLimit && i >= limit {
			return false
		}

		out = append(out, cleanSpaces(row.Text()))
		return true
	})

	return out, nil
}

// BuildURL returns the full url of given path
func (page *WebPage) BuildURL(pathURL string) (string, error) {
	u, err := url.Parse(page.baseURL)
	if err != nil {
		return "", err
	}

	if pathURL == "" {
		return u.String(), nil
	}

	urlPathParsed, err := url.Parse(pathURL)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(u.Path, urlPathParsed.Path)
	u.RawQuery = urlPathParsed.RawQuery

	return u.String(), nil
}

// GetRawTextForEachElement extracts the text of each element given in the selector until the given limit is surpassed.
// the returned output is not cleaned
func (page *WebPage) GetRawTextForEachElement(selector string, limit int) ([]string, error) {
	var (
		out     []string
		unLimit = limit <= 0
	)

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).EachWithBreak(func(i int, row *goquery.Selection) bool {
		if !unLimit && i >= limit {
			return false
		}

		out = append(out, row.Text())
		return true
	})

	return out, nil
}

// GetTextForEachElement extracts the text of each element given in the selector until the given limit is surpassed.
func (page *WebPage) GetTextForEachElement(selector string, limit int) ([]string, error) {
	var (
		out     []string
		unLimit = limit <= 0
	)

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).EachWithBreak(func(i int, row *goquery.Selection) bool {
		if !unLimit && i >= limit {
			return false
		}

		out = append(out, cleanText(row))
		return true
	})

	return out, nil
}

func cleanText(selection *goquery.Selection) string {
	txt := selection.Clone().Children().Remove().End().Text()
	return cleanSpaces(txt)
}

func (page *WebPage) checkIfLoaded() error {
	if page.doc == nil {
		return ErrPageNotLoaded
	}

	return nil
}

func cleanSpaces(txt string) string {
	txt = strings.TrimSpace(txt)
	return strings.Join(strings.Fields(txt), " ")
}

// GetAttr extract the value of a selected item
func (page *WebPage) GetAttr(selector, attr string) (string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return "", err
	}

	value, exist := page.doc.Find(selector).First().Attr(attr)
	if !exist {
		return "", ErrElementNotFound
	}

	return cleanSpaces(value), nil
}

// GetFullURL get the full URL of the page.
// The URL of the page only is modified with the Load method.
func (page *WebPage) GetFullURL() string {
	return page.fullURL
}

// GetAttrsWithOptions returns attribute values items that match the given options
func (page *WebPage) GetAttrsWithOptions(selector string, opts QueryOptions) ([]string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return nil, err
	}

	attrs := []string{}
	appendToAttrs := func(value string) bool {
		attrs = append(attrs, value)
		return opts.Limit > 0 && len(attrs) >= opts.Limit
	}

	page.doc.Find(selector).EachWithBreak(getAttrsWithOptions(appendToAttrs, opts))

	return attrs, nil
}

func getAttrsWithOptions(appendToAttrs appendFunc, opts QueryOptions) forEachFunc {
	return func(i int, s *goquery.Selection) bool {
		if skip(i, opts.Skip) {
			return true
		}

		if textFilter(getText(s, opts.Clean), opts.TextFilter) {
			return true
		}

		s = getSubSelector(s, opts.SubSelector)

		value, ok := s.Attr(opts.Attr)
		if !ok {
			return true
		}

		limitReached := appendToAttrs(value)

		return !limitReached // continue only if limit hasn't been reached
	}
}

func skip(i, skip int) bool {
	return skip > 0 && i < skip
}

func getText(s *goquery.Selection, clean bool) string {
	if clean {
		return cleanText(s)
	}

	return cleanSpaces(s.Text())
}

func textFilter(txt string, txtFilter textFilterFunc) bool {
	return txtFilter != nil && !txtFilter(txt)
}

func getSubSelector(s *goquery.Selection, subSelector string) *goquery.Selection {
	if subSelector == "" {
		return s
	}

	return s.Find(subSelector).First()
}

// GetRawBody extract the raw text of body.
func (page *WebPage) GetRawBody() (string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return "", err
	}

	return page.doc.Text(), nil
}

// GetHTMLRemovingSelectors return all text in html format
// removing children sent in selectors
func (page *WebPage) GetHTMLRemovingSelectors(selector string, childrenSelectors []string) (string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return "", err
	}

	selection := page.doc.Find(selector)

	for _, childrenSelector := range childrenSelectors {
		selection.Find(childrenSelector).Remove().End()
	}

	html, err := selection.Html()
	if err != nil {
		return "", err
	}

	return cleanSpaces(html), nil
}

// GetHTML return all text in html format
func (page *WebPage) GetHTML() (string, error) {
	if err := page.checkIfLoaded(); err != nil {
		return "", err
	}

	html, err := page.doc.Html()
	if err != nil {
		return "", err
	}

	return html, nil
}

// GetAttrTextForEachElement extracts the attr of each element given in the selector until the given limit is surpassed.
func (page *WebPage) GetAttrTextForEachElement(selector string, attr string, limit int) ([]string, error) {
	var (
		out     []string
		unLimit = limit <= 0
	)

	if limit > 0 {
		out = make([]string, 0, limit)
	}

	if err := page.checkIfLoaded(); err != nil {
		return out, err
	}

	page.doc.Find(selector).EachWithBreak(func(i int, row *goquery.Selection) bool {
		if !unLimit && i >= limit {
			return false
		}

		value, exist := row.Attr(attr)
		if exist {
			out = append(out, value)
		}

		return true
	})

	return out, nil
}

// GetElementsByAttributes returns the HTML elements given a set of properties
func (page *WebPage) GetElementsByAttributes(elementTag string, attrsToFind map[string]string) []*goquery.Selection {
	foundElements := []*goquery.Selection{}

	page.doc.Find(elementTag).Each(func(n int, s *goquery.Selection) {
		for attrName, attrValue := range attrsToFind {
			attr, exists := s.Attr(attrName)
			if !exists || attr != attrValue {
				return
			}
		}

		foundElements = append(foundElements, s)
	})

	return foundElements
}

// GetParentElementWithDepth returns the parent of an element given the depth
func (page *WebPage) GetParentElementWithDepth(selection *goquery.Selection, parentDepth int) *goquery.Selection {
	parent := &goquery.Selection{}

	for i := 0; i < parentDepth; i++ {
		selection = page.doc.FindSelection(selection).Parent()

		if i == parentDepth-1 {
			parent = selection
		}
	}

	return parent
}

// GetNextSiblingElementWithDepth returns the siblings of the next elements given the depth
func (page *WebPage) GetNextSiblingElementWithDepth(selection *goquery.Selection, siblingDepth int) *goquery.Selection {
	sibling := &goquery.Selection{}

	for i := 0; i < siblingDepth; i++ {
		selection = page.doc.FindSelection(selection).Next()

		if i == siblingDepth-1 {
			sibling = selection
		}
	}

	return sibling
}

// GetPrevSiblingElementWithDepth returns the siblings of previous elements given the depth
func (page *WebPage) GetPrevSiblingElementWithDepth(selection *goquery.Selection, siblingDepth int) *goquery.Selection {
	sibling := &goquery.Selection{}

	for i := 0; i < siblingDepth; i++ {
		selection = page.doc.FindSelection(selection).Prev()

		if i == siblingDepth-1 {
			sibling = selection
		}
	}

	return sibling
}

// GetElementByAttributesFromSelection returns the HTML elements given a set of properties, a tag and a *goquery.Selection
func (page *WebPage) GetElementByAttributesFromSelection(selection *goquery.Selection, elementTag string, attrsToFind map[string]string) *goquery.Selection {
	foundElement := &goquery.Selection{}

	page.doc.FindSelection(selection).Find(elementTag).Each(func(n int, s *goquery.Selection) {
		for attrName, attrValue := range attrsToFind {
			attr, exists := s.Attr(attrName)
			if !exists || attr != attrValue {
				return
			}
		}

		foundElement = s
	})

	return foundElement
}

// GetTextFromSelection extracts text form a selected node
func (page *WebPage) GetTextFromSelection(selector *goquery.Selection) string {
	return cleanSpaces(page.doc.FindSelection(selector).Text())
}

// GetTablesFromSelection returns all of the tables from a goquery selection
func (page *WebPage) GetTablesFromSelection(selector *goquery.Selection) []*goquery.Selection {
	subtables := []*goquery.Selection{}

	selector.Find("table").Each(func(i int, s *goquery.Selection) {
		subtables = append(subtables, s)
	})

	return subtables
}

// GetTableDivDataFromSelection gets the table text from a table with text wrapped in divs
func (page *WebPage) GetTableDivDataFromSelection(table *goquery.Selection) [][]string {
	var out [][]string

	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		info := row.Find("td").Map(func(index int, selection *goquery.Selection) string {
			return removeSpaces(selection.ChildrenFiltered("div").Text())
		})

		if len(info) > 0 {
			out = append(out, info)
		}
	})

	return out
}

// GetTextFromSelectionTag extracts text form a selection element
func (page *WebPage) GetTextFromSelectionTag(selector *goquery.Selection, tag string) string {
	return cleanSpaces(page.doc.FindSelection(selector).Find(tag).First().Text())
}

func removeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// GetTableDataFromSelection gets the table text from a selected table
func (page *WebPage) GetTableDataFromSelection(table *goquery.Selection) [][]string {
	var out [][]string

	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		info := row.Find("td").Map(func(index int, selection *goquery.Selection) string {
			return removeSpaces(selection.Text())
		})

		if len(info) > 0 {
			out = append(out, info)
		}
	})

	return out
}

// GetElementsBySelector returns the HTML elements given a selector
func (page *WebPage) GetElementsBySelector(selector string) []*goquery.Selection {
	foundElements := []*goquery.Selection{}

	page.doc.Find(selector).Each(func(n int, s *goquery.Selection) {
		foundElements = append(foundElements, s)
	})

	return foundElements
}

// GetDoc gets goquery document. Use it when the above aux functions don't work for
// your implementations.
func (page *WebPage) GetDoc() *goquery.Document {
	return page.doc
}

// LogRequestBlocked logs request blocked for last page loaded
func (page *WebPage) LogRequestBlocked(ctx context.Context) error {
	return page.client.LogRequestBlocked(ctx, page.response)
}
