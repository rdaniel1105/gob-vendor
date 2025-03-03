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
		idUNSPSC       string
		description    string
		specifications string
		quantity       string
	)

	idUNSPSCElements := page.GetElementsBySelector(detailUNSPSCSelector)
	if len(idUNSPSCElements) > 0 {
		idUNSPSC = idUNSPSCElements[0].Text()
	}

	descriptionElements := page.GetElementsBySelector(detailDescriptionSelector)
	if len(descriptionElements) > 0 {
		description = descriptionElements[0].Text()
	}

	specificationsElements := page.GetElementsBySelector(detailSpecificationsSelector)
	if len(specificationsElements) > 0 {
		specifications = specificationsElements[0].Text()
	}

	quantityElements := page.GetElementsBySelector(detailQuantitySelector)
	if len(quantityElements) > 0 {
		quantity = quantityElements[0].Text()
	}

	startDate, err := getDate(generalData[4].Find(detailGeneralDataValueSelector).Text())
	if err != nil {
		return nil, err
	}

	receptionDate, err := getDate(generalData[5].Find(detailGeneralDataValueSelector).Text())
	if err != nil {
		return nil, err
	}

	endDate, err := getDate(generalData[6].Find(detailGeneralDataValueSelector).Text())
	if err != nil {
		return nil, err
	}

	detail := &Detail{
		ExpedientID:     generalData[0].Find(detailGeneralDataValueSelector).Text(),
		Entity:          generalData[1].Find(detailGeneralDataValueSelector).Text(),
		Unit:            generalData[2].Find(detailGeneralDataValueSelector).Text(),
		Object:          generalData[3].Find(detailGeneralDataValueSelector).Text(),
		StartDate:       startDate,
		ReceptionDate:   receptionDate,
		EndDate:         endDate,
		SourceType:      generalData[7].Find(detailGeneralDataValueSelector).Text(),
		Source:          generalData[8].Find(detailGeneralDataValueSelector).Text(),
		Category:        AdquisitionCategory(generalData[9].Find(detailGeneralDataValueSelector).Text()),
		Stage:           AdquisitionStage(generalData[10].Find(detailGeneralDataValueSelector).Text()),
		AcquisitionType: AdquisitionType(generalData[11].Find(detailGeneralDataValueSelector).Text()),
		ReceptionPlace:  generalData[12].Find(detailGeneralDataValueSelector).Text(),
		Currency:        CurrencyType(generalData[13].Find(detailCurrencySelector).Text()),
		TenderCost:      strToFloat(generalData[13].Find(detailValueSelector).Text()),
		ContactName:     generalData[14].Find(detailContactNameSelector).Text(),
		ContactPhone:    generalData[14].Find(detailContactPhoneSelector).Text(),
		ContactEmail:    generalData[14].Find(detailContactEmailSelector).Text(),
		IDUNSPSC:        idUNSPSC,
		Description:     description,
		Specifications:  specifications,
		Quantity:        strToFloat(quantity),
		Page:            pageNumber,
	}

	return detail, nil
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
