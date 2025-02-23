package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/webscraper"
)

type AdquisitionValueType string
type AdquisitionStatusType string
type AdquisitionModalType string

const (
	AdquisitionValueAllType         AdquisitionValueType = "(Todas)"
	ServicesAndGoodsAdquisitionType AdquisitionValueType = "Suministro de Bienes y/o Servicios"
	WorksAdquisitionType            AdquisitionValueType = "Obras"
	ConsultingAdquisitionType       AdquisitionValueType = "Consultoria"

	AdquisitionStatusAllType            AdquisitionStatusType = "(Todas)"
	AdquisitionStatusElaborationType    AdquisitionStatusType = "Elaboración"
	AdquisitionStatusReviewedType       AdquisitionStatusType = "Revisado"
	AdquisitionStatusReceivedOffersType AdquisitionStatusType = "Recepción de Ofertas"
	AdquisitionStatusEvaluationType     AdquisitionStatusType = "Evaluación"
	AdquisitionStatusAwardedType        AdquisitionStatusType = "Adjudicado"
	AdquisitionStatusDesertedType       AdquisitionStatusType = "Desierto"
	AdquisitionStatusFailedType         AdquisitionStatusType = "Fracasados"

	AdquisitionModalAllType                 AdquisitionModalType = "(Todas)"
	AdquisitionModalLicitationOrContestType AdquisitionModalType = "Licitación o Concurso"
	AdquisitionModalMinorPurchaseType       AdquisitionModalType = "Compra Menor"
	AdquisitionModalPrequalificationType    AdquisitionModalType = "Precalificación"

	ViewStateGeneratorSelector = "#__VIEWSTATEGENERATOR"
	ViewStateSelector          = "#__VIEWSTATE"
	EventValidationSelector    = "#__EVENTVALIDATION"

	viewStateGeneratorParam = "__VIEWSTATEGENERATOR"
	viewStateParam          = "__VIEWSTATE"
	eventValidationParam    = "__EVENTVALIDATION"

	lastPageEventArgument = "Page$Last"
	paginationEventTarget = "ctl00$cphCuerpo$gvResultados"

	pagesSelector       = "#ctl00_cphCuerpo_gvResultados > tbody > tr:nth-child(%d) > td > table > tbody > tr > td"
	detailsURLsSelector = "#ctl00_cphCuerpo_gvResultados > tbody > tr > td:nth-child(5) > a"

	pageNumberExceeding30 = 32

	additionalTableRows = 2

	detailsURLAttr = "href"
	paramsAttr     = "value"
)

var (
	ErrInvalidAdquisitionType   = errors.New("invalid adquisition type")
	ErrInvalidAdquisitionStatus = errors.New("invalid adquisition status")
	ErrInvalidAdquisitionModal  = errors.New("invalid adquisition modal")

	BaseURL    = "http://sicc.honducompras.gob.hn/HC/procesos/busquedahistorico.aspx"
	DetailsURL = "http://sicc.honducompras.gob.hn/HC/procesos/%s"

	adquisitionValues = map[AdquisitionValueType]string{
		AdquisitionValueAllType:         "0",
		ServicesAndGoodsAdquisitionType: "1",
		WorksAdquisitionType:            "2",
		ConsultingAdquisitionType:       "3",
	}

	adquisitionStatusValues = map[AdquisitionStatusType]string{
		AdquisitionStatusAllType:            "0",
		AdquisitionStatusElaborationType:    "1",
		AdquisitionStatusReviewedType:       "2",
		AdquisitionStatusReceivedOffersType: "3",
		AdquisitionStatusEvaluationType:     "4",
		AdquisitionStatusAwardedType:        "5",
		AdquisitionStatusDesertedType:       "6",
		AdquisitionStatusFailedType:         "7",
	}

	adquisitionModalValues = map[AdquisitionModalType]string{
		AdquisitionModalAllType:                 "0",
		AdquisitionModalLicitationOrContestType: "1",
		AdquisitionModalMinorPurchaseType:       "2",
		AdquisitionModalPrequalificationType:    "3",
	}
)

func NewFetcher(adquisitionType AdquisitionValueType, adquisitionStatus AdquisitionStatusType, adquisitionModal AdquisitionModalType, client *client.Client, logger *slog.Logger) (*Fetcher, error) {
	adquisitionValue, ok := adquisitionValues[adquisitionType]
	if !ok {
		return nil, ErrInvalidAdquisitionType
	}

	adquisitionStatusValue, ok := adquisitionStatusValues[adquisitionStatus]
	if !ok {
		return nil, ErrInvalidAdquisitionStatus
	}

	adquisitionModalValue, ok := adquisitionModalValues[adquisitionModal]
	if !ok {
		return nil, ErrInvalidAdquisitionModal
	}

	return &Fetcher{
		adquisitionType:   adquisitionValue,
		adquisitionStatus: adquisitionStatusValue,
		adquisitionModal:  adquisitionModalValue,
		client:            client,
		logger:            logger,
		pages:             1,
		Report:            &Report{},
	}, nil
}

type Fetcher struct {
	adquisitionType    string
	adquisitionStatus  string
	adquisitionModal   string
	client             *client.Client
	logger             *slog.Logger
	pages              int64
	Report             *Report
	viewStateGenerator string
	viewState          string
	eventValidation    string
}

func (f *Fetcher) Execute(ctx context.Context) error {
	page, err := f.doInintialRequest(ctx)
	if err != nil {
		return err
	}

	err = f.setParamsFromPage(page)
	if err != nil {
		return err
	}

	resultsPage, err := f.doSearchRequest(ctx)
	if err != nil {
		return err
	}

	err = f.processDetailsPage(ctx, resultsPage)
	if err != nil {
		return err
	}

	return nil
}

func (f *Fetcher) doInintialRequest(ctx context.Context) (*webscraper.WebPage, error) {
	page := webscraper.NewWebPage(BaseURL, f.client, f.logger)

	response, err := f.client.Get(ctx, BaseURL, nil, nil)
	if err != nil {
		return nil, err
	}

	if err := page.LoadFromResponse(ctx, response); err != nil {
		return nil, err
	}

	return &page, nil
}

func (f *Fetcher) doSearchRequest(ctx context.Context) (*webscraper.WebPage, error) {
	params := f.getPostRequestParams("", "")

	response, err := f.client.Post(ctx, BaseURL, nil, params)
	if err != nil {
		return nil, err
	}

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

func (f *Fetcher) processDetailsPage(ctx context.Context, page *webscraper.WebPage) error {
	details, err := page.GetAttrs(detailsURLsSelector, detailsURLAttr)
	if err != nil {
		return err
	}

	if len(details) >= 30 {
		f.pages, err = f.getNumberOfPages(ctx, page, pageNumberExceeding30)
		if err != nil {
			return err
		}
	}

	strBuilder := strings.Builder{}

	for _, result := range details {
		strBuilder.WriteString(result)
		strBuilder.WriteString("\n")
	}

	html := strBuilder.String()

	err = os.WriteFile("page.txt", []byte(html), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (f *Fetcher) getPostRequestParams(eventArgument, eventTarget string) url.Values {
	params := url.Values{}

	params.Set("__EVENTVALIDATION", f.eventValidation)
	params.Set("__VIEWSTATE", f.viewState)
	params.Set("__VIEWSTATEGENERATOR", f.viewStateGenerator)
	params.Set("ctl00_tkSM_HiddenField", "")
	params.Set("__EVENTTARGET", eventTarget)
	params.Set("__EVENTARGUMENT", eventArgument)
	params.Set("__LASTFOCUS", "")
	params.Set("ctl00$cphCuerpo$wpParametros_hidden", "")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlEntidades", "0")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlUC", "0")
	params.Set("ctl00_cphCuerpo_wpParametros_wdInicio_DrpPnl1_DP_CAL_ID_1", "%3Cx%20PostData%3D%222025x2x-1x-1x-1%22%3E%3C/x%3E")
	params.Set("ctl00$cphCuerpo$wpParametros$wdInicio_hidden", "")
	params.Set("ctl00_cphCuerpo_wpParametros_wdInicio_input", "(Todas)")
	params.Set("ctl00_cphCuerpo_wpParametros_wdFin_DrpPnl2_DP_CAL_ID_2", "%3Cx%20PostData%3D%222025x2x-1x-1x-1%22%3E%3C/x%3E")
	params.Set("ctl00_cphCuerpo_wpParametros_wdFin_input", "(Todas)")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlModalidad", f.adquisitionModal)
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
	page, err := f.doLastPageRequest(ctx)
	if err != nil {
		return 0, err
	}

	details, err := page.GetAttrs(detailsURLsSelector, detailsURLAttr)
	if err != nil {
		return 0, err
	}

	pages := page.GetElementsBySelector(fmt.Sprintf(pagesSelector, len(details)+additionalTableRows))

	return strconv.ParseInt(pages[len(pages)-1].Text(), 10, 64)
}

func (f *Fetcher) doLastPageRequest(ctx context.Context) (*webscraper.WebPage, error) {
	params := f.getPostRequestParams(lastPageEventArgument, paginationEventTarget)

	response, err := f.client.Post(ctx, BaseURL, nil, params)
	if err != nil {
		return nil, err
	}

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

func (f *Fetcher) setParamsFromPage(page *webscraper.WebPage) error {
	viewStateGenerator, err := page.GetAttr(ViewStateGeneratorSelector, paramsAttr)
	if err != nil {
		return err
	}

	f.viewStateGenerator = viewStateGenerator

	viewState, err := page.GetAttr(ViewStateSelector, paramsAttr)
	if err != nil {
		return err
	}

	f.viewState = viewState

	eventValidation, err := page.GetAttr(EventValidationSelector, paramsAttr)
	if err != nil {
		return err
	}

	f.eventValidation = eventValidation

	return nil
}
