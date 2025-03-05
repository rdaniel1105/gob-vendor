package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/env"
	"github.com/rdaniel1105/gob-vendor/webscraper"
)

const (
	ViewStateGeneratorSelector = "#__VIEWSTATEGENERATOR"
	ViewStateSelector          = "#__VIEWSTATE"
	EventValidationSelector    = "#__EVENTVALIDATION"

	viewStateGeneratorParam = "__VIEWSTATEGENERATOR"
	viewStateParam          = "__VIEWSTATE"
	eventValidationParam    = "__EVENTVALIDATION"

	lastPageEventArgument  = "Page$Last"
	firstPageEventArgument = "Page$First"

	// PaginationEventTarget is the target for the pagination event
	PaginationEventTarget = "ctl00$cphCuerpo$gvResultados"

	pagesSelector           = "#ctl00_cphCuerpo_gvResultados > tbody > tr:nth-child(%d) > td > table > tbody > tr > td"
	detailsURLsSelector     = "td:nth-child(5) > a"
	detailsSelector         = "#ctl00_cphCuerpo_gvResultados > tbody > tr"
	completeDetailsSelector = "#ctl00_cphCuerpo_gvResultados > tbody > tr > td:nth-child(5) > a"

	expedientIDSelector = "td:nth-child(1) > span:nth-child(2)"
	entitySelector      = "td:nth-child(1) > span:nth-child(5)"
	unitSelector        = "td:nth-child(1) > span:nth-child(8)"
	objectSelector      = "td:nth-child(1) > span:nth-child(12)"

	startDateSelector = "td:nth-child(4) > span:nth-child(2)"
	endDateSelector   = "td:nth-child(4) > span:nth-child(5)"

	pageNumberExceeding30 = 32

	additionalTableRows = 2

	detailsURLAttr = "href"
	paramsAttr     = "value"
)

var (
	ErrInvalidAdquisitionType   = errors.New("invalid adquisition type")
	ErrInvalidAdquisitionStatus = errors.New("invalid adquisition status")
	ErrInvalidAdquisitionModal  = errors.New("invalid adquisition modal")
	ErrNotFound                 = errors.New("details not found")

	errNoPagesFound = errors.New("no pages found")

	BaseURL    = "http://sicc.honducompras.gob.hn/HC/procesos/busquedahistorico.aspx"
	DetailsURL = "http://sicc.honducompras.gob.hn/HC/procesos/"

	adquisitionValues = map[AdquisitionType]string{
		AdquisitionTypeAll:              "0",
		AdquisitionTypeServicesAndGoods: "1",
		AdquisitionTypeWorks:            "2",
		AdquisitionTypeConsulting:       "3",
	}

	adquisitionStatusValues = map[AdquisitionStage]string{
		AdquisitionStageAll:                "0",
		AdquisitionStageElaborationType:    "1",
		AdquisitionStageReviewedType:       "2",
		AdquisitionStageReceivedOffersType: "3",
		AdquisitionStageEvaluationType:     "4",
		AdquisitionStageAwardedType:        "5",
		AdquisitionStageDesertedType:       "6",
		AdquisitionStageFailedType:         "7",
	}

	adquisitionCategoryValues = map[AdquisitionCategory]string{
		AdquisitionCategoryAll:                     "0",
		AdquisitionCategoryLicitationOrContestType: "1",
		AdquisitionCategoryMinorPurchaseType:       "2",
		AdquisitionCategoryPrequalificationType:    "3",
	}

	// PaginationEventArgument is the argument for the pagination event
	PaginationEventArgument = "Page$%d"

	setCredentialsOnce sync.Once

	admin         string
	password      string
	zincSearchURL string
)

type Fetcher struct {
	adquisitionType     string
	adquisitionStatus   string
	adquisitionCategory string
	client              *client.Client
	logger              *slog.Logger
	pages               int64
	pageNumber          int64
	ViewStateGenerator  string
	ViewState           string
	EventValidation     string
	eventTarget         string
	eventArgument       string
	Cookies             string
	SessionStateChan    chan SessionState
}

type SessionState struct {
	ViewState          string
	ViewStateGenerator string
	EventValidation    string
}

func NewFetcher(
	eventTarget string,
	eventArgument string,
	pageNumber int64,
	adquisitionType AdquisitionType,
	adquisitionStage AdquisitionStage,
	adquisitionCategory AdquisitionCategory,
	client *client.Client,
	logger *slog.Logger,
) (*Fetcher, error) {
	adquisitionValue, ok := adquisitionValues[adquisitionType]
	if !ok {
		return nil, ErrInvalidAdquisitionType
	}

	adquisitionStatusValue, ok := adquisitionStatusValues[adquisitionStage]
	if !ok {
		return nil, ErrInvalidAdquisitionStatus
	}

	adquisitionCategoryValue, ok := adquisitionCategoryValues[adquisitionCategory]
	if !ok {
		return nil, ErrInvalidAdquisitionModal
	}

	return &Fetcher{
		eventTarget:         eventTarget,
		eventArgument:       eventArgument,
		adquisitionType:     adquisitionValue,
		adquisitionStatus:   adquisitionStatusValue,
		adquisitionCategory: adquisitionCategoryValue,
		client:              client,
		logger:              logger,
		pageNumber:          pageNumber,
	}, nil
}

func (f *Fetcher) GetPages() int64 {
	return f.pages
}

func setCredentials() {
	zincSearchURL = env.GetString("ZINCSEARCH_URL", "")

	admin = env.GetString("ZINCSEARCH_USERNAME", "")
	password = env.GetString("ZINCSEARCH_PASSWORD", "")
}

func (f *Fetcher) Execute(ctx context.Context) error {
	setCredentialsOnce.Do(setCredentials)

	var resultsPage *webscraper.WebPage
	var err error

	if f.pageNumber == 1 {
		err = f.doInintialRequest(ctx)
		if err != nil {
			return err
		}

		resultsPage, err = f.doSearchRequest(ctx)
		if err != nil {
			return err
		}
	} else {
		if f.pageNumber == 2 {
			err = f.executeFirstPageRequest(ctx)
			if err != nil {
				return err
			}
		}

		resultsPage, err = f.doPaginationRequest(ctx, fmt.Sprintf(PaginationEventArgument, f.pageNumber))
		if err != nil {
			return err
		}
	}

	details, err := f.getPageDetails(ctx, resultsPage, f.pageNumber)
	if err != nil {
		return err
	}

	return sendDetails(details)
}

func (f *Fetcher) doInintialRequest(ctx context.Context) error {
	page := webscraper.NewWebPage(BaseURL, f.client, f.logger)

	response, err := f.client.Get(ctx, BaseURL, nil, nil)
	if err != nil {
		return err
	}

	if err := page.LoadFromResponse(ctx, response); err != nil {
		return err
	}

	if err := f.setParamsFromPage(&page); err != nil {
		return err
	}

	return nil
}

func (f *Fetcher) doSearchRequest(ctx context.Context) (*webscraper.WebPage, error) {
	params := f.getPostRequestParams("", "")

	response, err := f.client.Post(ctx, BaseURL, nil, params)
	if err != nil {
		return nil, err
	}

	f.Cookies = f.client.GetCookies(BaseURL)

	page := webscraper.NewWebPage(BaseURL, f.client, f.logger)

	if err := page.LoadFromResponse(ctx, response); err != nil {
		return nil, err
	}

	err = f.setParamsFromPage(&page)
	if err != nil {
		return nil, err
	}

	return &page, nil
}

func (f *Fetcher) getPageDetails(ctx context.Context, page *webscraper.WebPage, pageNumber int64) ([]*ListItem, error) {
	tableSelection := page.GetElementsBySelector(detailsSelector)

	if len(tableSelection) == 0 {
		return nil, ErrNotFound
	}

	detailsURLs := make([]*ListItem, 0, len(tableSelection)-1)

	for _, detail := range tableSelection[:len(tableSelection)-1] {
		expedientID := detail.Find(expedientIDSelector).Text()

		url, ok := detail.Find(detailsURLsSelector).Attr(detailsURLAttr)
		if !ok {
			f.logger.Error("details_url_not_found", "page", pageNumber, "expedient", expedientID)
			continue
		}

		entity := detail.Find(entitySelector).Text()
		unit := detail.Find(unitSelector).Text()
		object := detail.Find(objectSelector).Text()
		startDate := detail.Find(startDateSelector).Text()
		endDate := detail.Find(endDateSelector).Text()

		detailsURLs = append(detailsURLs, NewDetail(
			expedientID,
			entity,
			unit,
			object,
			startDate,
			endDate,
			url,
			f.ViewState,
			f.EventValidation,
			f.ViewStateGenerator,
			pageNumber,
		))
	}

	if len(detailsURLs) == 0 {
		return nil, ErrNotFound
	}

	if len(detailsURLs) >= 30 && pageNumber == 1 {
		var err error

		f.pages, err = f.getNumberOfPages(ctx, page, pageNumberExceeding30)
		if errors.Is(err, errNoPagesFound) {
			f.logger.Info("no_pages_found")

			return nil, nil
		}

		if err != nil {
			return nil, err
		}
	}

	return detailsURLs, nil
}

func (f *Fetcher) getPostRequestParams(eventArgument, eventTarget string) url.Values {
	params := url.Values{}

	params.Set("ctl00_tkSM_HiddenField", "")
	params.Set("__EVENTTARGET", eventTarget)
	params.Set("__EVENTARGUMENT", eventArgument)
	params.Set("__LASTFOCUS", "")
	params.Set("__VIEWSTATE", f.ViewState)
	params.Set("__VIEWSTATEGENERATOR", f.ViewStateGenerator)
	params.Set("__EVENTVALIDATION", f.EventValidation)
	params.Set("ctl00$cphCuerpo$wpParametros_hidden", "")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlEntidades", "0")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlUC", "0")
	params.Set("ctl00_cphCuerpo_wpParametros_wdInicio_DrpPnl1_DP_CAL_ID_1", "%3Cx%20PostData%3D%222025x2x-1x-1x-1%22%3E%3C/x%3E")
	params.Set("ctl00$cphCuerpo$wpParametros$wdInicio_hidden", "")
	params.Set("ctl00_cphCuerpo_wpParametros_wdInicio_input", "(Todas)")
	params.Set("ctl00_cphCuerpo_wpParametros_wdFin_DrpPnl2_DP_CAL_ID_2", "%3Cx%20PostData%3D%222025x2x-1x-1x-1%22%3E%3C/x%3E")
	params.Set("ctl00$cphCuerpo$wpParametros$wdFin_hidden", "")
	params.Set("ctl00_cphCuerpo_wpParametros_wdFin_input", "(Todas)")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlModalidad", f.adquisitionCategory)
	params.Set("ctl00$cphCuerpo$wpParametros$ddlTipoAdquisicion", f.adquisitionType)
	params.Set("ctl00$cphCuerpo$wpParametros$ddlEtapas", f.adquisitionStatus)
	params.Set("ctl00$cphCuerpo$wpParametros$ddlFteFinanciamiento", "0")
	params.Set("ctl00$cphCuerpo$wpParametros$txtObjetoCompra", "")
	params.Set("ctl00$cphCuerpo$wpParametros$txtExpediente", "")

	if eventArgument == "" {
		params.Set("ctl00$cphCuerpo$wpParametros$btnBuscar", "Buscar")
	}

	params.Set("ctl00$_IG_CSS_LINKS_", "../StyleSheet.css|../ig_res/Default/ig_calendar.css|../ig_res/Default/ig_datechooser.css|../ig_res/Default/ig_panel.css|../ig_res/Default/ig_shared.css")

	return params
}

func (f *Fetcher) getNumberOfPages(ctx context.Context, page *webscraper.WebPage, detailsNumber int64) (int64, error) {
	pages := page.GetElementsBySelector(fmt.Sprintf(pagesSelector, detailsNumber))
	if len(pages) == 0 {
		return 0, nil
	}

	if len(pages) > 20 {
		return f.executeLastPageRequest(ctx)
	}

	return int64(len(pages)), nil
}

func (f *Fetcher) executeLastPageRequest(ctx context.Context) (int64, error) {
	page, err := f.doPaginationRequest(ctx, lastPageEventArgument)
	if err != nil {
		return 0, err
	}

	details, err := page.GetAttrs(completeDetailsSelector, detailsURLAttr)
	if err != nil {
		return 0, err
	}

	pages := page.GetElementsBySelector(fmt.Sprintf(pagesSelector, len(details)+additionalTableRows))
	if len(pages) == 0 {
		return 0, errNoPagesFound
	}

	return strconv.ParseInt(pages[len(pages)-1].Text(), 10, 64)
}

func (f *Fetcher) executeFirstPageRequest(ctx context.Context) error {
	_, err := f.doPaginationRequest(ctx, firstPageEventArgument)
	if err != nil {
		return err
	}

	return err
}

func (f *Fetcher) doPaginationRequest(ctx context.Context, eventArgument string) (*webscraper.WebPage, error) {
	params := f.getPostRequestParams(eventArgument, PaginationEventTarget)

	headers := http.Header{}
	headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	headers.Set("Accept-Encoding", "gzip, deflate")
	headers.Set("Accept-Language", "en-US,en;q=0.5")
	headers.Set("Cache-Control", "max-age=0")
	headers.Set("Connection", "keep-alive")
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("Origin", "http://sicc.honducompras.gob.hn")
	headers.Set("Referer", BaseURL)
	headers.Set("Sec-GPC", "1")
	headers.Set("Upgrade-Insecure-Requests", "1")
	headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")

	response, err := f.client.Post(ctx, BaseURL, headers, params)
	if err != nil {
		return nil, err
	}

	f.Cookies = f.client.GetCookies(BaseURL)

	page := webscraper.NewWebPage(BaseURL, f.client, f.logger)

	if err := page.LoadFromResponse(ctx, response); err != nil {
		return nil, err
	}

	err = f.setParamsFromPage(&page)
	if err != nil {
		return nil, err
	}

	if f.pageNumber != 1 && f.SessionStateChan != nil {
		select {
		case f.SessionStateChan <- SessionState{
			ViewState:          f.ViewState,
			ViewStateGenerator: f.ViewStateGenerator,
			EventValidation:    f.EventValidation,
		}:
		default:
			f.logger.Debug("could not send session state to channel")
		}
	}

	return &page, nil
}

func (f *Fetcher) setParamsFromPage(page *webscraper.WebPage) error {
	viewStateGenerator, err := page.GetAttr(ViewStateGeneratorSelector, paramsAttr)
	if err != nil {
		return err
	}

	viewState, err := page.GetAttr(ViewStateSelector, paramsAttr)
	if err != nil {
		return err
	}

	eventValidation, err := page.GetAttr(EventValidationSelector, paramsAttr)
	if err != nil {
		return err
	}

	f.ViewStateGenerator = viewStateGenerator
	f.ViewState = viewState
	f.EventValidation = eventValidation

	return nil
}
