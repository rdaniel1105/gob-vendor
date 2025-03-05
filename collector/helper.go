package collector

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/webscraper"
)

const (
	generalTableSelector           = "#ctl00_cphCuerpo_dvProceso > tbody > tr"
	detailGeneralDataValueSelector = "td:nth-child(2)"

	detailCurrencySelector     = "td:nth-child(2) > span:nth-child(1)"
	detailValueSelector        = "td:nth-child(2) > span:nth-child(2)"
	detailContactNameSelector  = "td:nth-child(2) > span:nth-child(1)"
	detailContactPhoneSelector = "td:nth-child(2) > span:nth-child(3)"
	detailContactEmailSelector = "td:nth-child(2) > a"

	detailUNSPSCSelector         = "#x\\:1955457169\\.13\\:adr\\:0\\:tag\\: > td:nth-child(1)"
	detailDescriptionSelector    = "#x\\:1955457169\\.13\\:adr\\:0\\:tag\\: > td:nth-child(2)"
	detailSpecificationsSelector = "#x\\:1955457169\\.13\\:adr\\:0\\:tag\\: > td:nth-child(3)"
	detailQuantitySelector       = "#x\\:1955457169\\.13\\:adr\\:0\\:tag\\: > td:nth-child(4)"

	dateLayout = "02/01/2006 03:04:05 PM"
)

var (
	maxDetailsConcurrency = runtime.NumCPU()
)

func NewDetail(
	expedientID,
	entity,
	unit,
	object,
	startDate,
	endDate,
	link,
	viewState,
	eventValidation,
	viewStateGenerator string,
	page int64,
) *ListItem {
	return &ListItem{
		ExpedientID:        expedientID,
		Entity:             entity,
		Unit:               unit,
		Object:             object,
		StartDate:          startDate,
		EndDate:            endDate,
		Link:               link,
		viewState:          viewState,
		eventValidation:    eventValidation,
		viewStateGenerator: viewStateGenerator,
		Page:               page,
	}
}

func (d *ListItem) getDetailToIndex() (string, error) {
	url := DetailsURL + d.Link

	newClient, err := client.New()
	if err != nil {
		return "", err
	}

	response, err := newClient.Get(context.Background(), url, nil, nil)
	if err != nil {
		slog.Error("error getting detail", "error", err)
		return "", err
	}

	page := webscraper.NewWebPage(url, newClient, slog.Default())

	err = page.LoadFromResponse(context.Background(), response)
	if err != nil {
		slog.Error("error loading detail", "error", err)
		return "", err
	}

	detail, err := setData(&page, d.Page)
	if err != nil {
		slog.Error("error setting detail", "error", err)
		return "", err
	}

	data, err := json.Marshal(detail)
	if err != nil {
		slog.Error("error marshalling detail", "error", err)
		return "", err
	}

	return string(data), nil
}

func sendDetails(details []*ListItem) error {
	wg := sync.WaitGroup{}

	errs := make([]error, 0, len(details))

	semaphore := make(chan struct{}, maxDetailsConcurrency)
	errChan := make(chan error)
	detailSender := make(chan string)

	wg.Add(1)
	go detailsChunkSender(detailSender, &wg)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var detailWg sync.WaitGroup

		for _, detail := range details {
			semaphore <- struct{}{}
			detailWg.Add(1)

			go func(d *ListItem) {
				defer func() {
					<-semaphore
					detailWg.Done()
				}()

				data, err := d.getDetailToIndex()
				if err != nil {
					errChan <- err
					return
				}

				detailSender <- data
			}(detail)
		}

		detailWg.Wait()

		close(detailSender)
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		errs = append(errs, err)
	}

	finalErr := errors.Join(errs...)

	if finalErr != nil {
		return finalErr
	}

	return nil
}

func setData(page *webscraper.WebPage, pageNumber int64) (*Detail, error) {
	generalData := page.GetElementsBySelector(generalTableSelector)

	var (
		idUNSPSCs      []string
		descriptions   []string
		specifications []string
		quantities     []string
	)

	idUNSPSCElements := page.GetElementsBySelector(detailUNSPSCSelector)
	if len(idUNSPSCElements) > 0 {
		idUNSPSCs = appendGoQueryStrSelections(idUNSPSCElements)
	}

	descriptionElements := page.GetElementsBySelector(detailDescriptionSelector)
	if len(descriptionElements) > 0 {
		descriptions = appendGoQueryStrSelections(descriptionElements)
	}

	specificationsElements := page.GetElementsBySelector(detailSpecificationsSelector)
	if len(specificationsElements) > 0 {
		specifications = appendGoQueryStrSelections(specificationsElements)
	}

	quantityElements := page.GetElementsBySelector(detailQuantitySelector)
	if len(quantityElements) > 0 {
		quantities = appendGoQueryStrSelections(quantityElements)
	}

	startDate, err := getDate(getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 4))
	if err != nil {
		return nil, err
	}

	receptionDate, err := getDate(getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 5))
	if err != nil {
		return nil, err
	}

	endDate, err := getDate(getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 6))
	if err != nil {
		return nil, err
	}

	detail := &Detail{
		ExpedientID:     getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 0),
		Entity:          getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 1),
		Unit:            getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 2),
		Object:          getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 3),
		StartDate:       startDate,
		ReceptionDate:   receptionDate,
		EndDate:         endDate,
		SourceType:      getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 7),
		Source:          getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 8),
		Category:        AdquisitionCategory(getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 9)),
		Stage:           AdquisitionStage(getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 10)),
		AcquisitionType: AdquisitionType(getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 11)),
		ReceptionPlace:  getGeneralDataFieldText(generalData, detailGeneralDataValueSelector, 12),
		Currency:        CurrencyType(getGeneralDataFieldText(generalData, detailCurrencySelector, 13)),
		TenderCost:      strToFloat(getGeneralDataFieldText(generalData, detailValueSelector, 13)),
		ContactName:     getGeneralDataFieldText(generalData, detailContactNameSelector, 14),
		ContactPhone:    getGeneralDataFieldText(generalData, detailContactPhoneSelector, 14),
		ContactEmail:    getGeneralDataFieldText(generalData, detailContactEmailSelector, 14),
		IDUNSPSCs:       idUNSPSCs,
		Descriptions:    descriptions,
		Specifications:  specifications,
		Quantities:      convertStrToFloatArray(quantities),
		Page:            pageNumber,
	}

	return detail, nil
}

func appendGoQueryStrSelections(selections []*goquery.Selection) []string {
	results := make([]string, 0, len(selections))

	for _, selection := range selections {
		results = append(results, selection.Text())
	}

	return results
}

func convertStrToFloatArray(strArray []string) []float64 {
	results := make([]float64, 0, len(strArray))

	for _, str := range strArray {
		results = append(results, strToFloat(str))
	}

	return results
}

func getGeneralDataFieldText(generalData []*goquery.Selection, selector string, index int) string {
	if len(generalData) <= index {
		return ""
	}

	element := generalData[index].Find(selector)

	return element.Text()
}

func strToFloat(s string) float64 {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	if value < 0 {
		return 0
	}

	return value
}

func getDate(date string) (time.Time, error) {
	date = replaceMeridian(date)

	return time.Parse(dateLayout, date)
}

func replaceMeridian(s string) string {
	s = strings.ReplaceAll(s, "a.m.", "AM")
	s = strings.ReplaceAll(s, "p.m.", "PM")

	return s
}
