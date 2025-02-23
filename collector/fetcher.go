package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/webscraper"
)

type AdquisitionValueType string
type AdquisitionStatusType string
type AdquisitionModalType string

const (
	AdquisitionValueAllType         AdquisitionValueType = "0"
	ServicesAndGoodsAdquisitionType AdquisitionValueType = "1"
	WorksAdquisitionType            AdquisitionValueType = "2"
	ConsultingAdquisitionType       AdquisitionValueType = "3"

	AdquisitionStatusAllType            AdquisitionStatusType = "0"
	AdquisitionStatusElaborationType    AdquisitionStatusType = "1"
	AdquisitionStatusReviewedType       AdquisitionStatusType = "2"
	AdquisitionStatusReceivedOffersType AdquisitionStatusType = "3"
	AdquisitionStatusEvaluationType     AdquisitionStatusType = "4"
	AdquisitionStatusAwardedType        AdquisitionStatusType = "5"
	AdquisitionStatusDesertedType       AdquisitionStatusType = "6"
	AdquisitionStatusFailedType         AdquisitionStatusType = "7"

	AdquisitionModalAllType                 AdquisitionModalType = "0"
	AdquisitionModalLicitationOrContestType AdquisitionModalType = "1"
	AdquisitionModalMinorPurchaseType       AdquisitionModalType = "2"
	AdquisitionModalPrequalificationType    AdquisitionModalType = "3"

	ViewStateGeneratorAttr = "#__VIEWSTATEGENERATOR"
	ViewStateAttr          = "#__VIEWSTATE"
	EventValidationAttr    = "#__EVENTVALIDATION"
)

var (
	ErrInvalidAdquisitionType   = errors.New("invalid adquisition type")
	ErrInvalidAdquisitionStatus = errors.New("invalid adquisition status")
	ErrInvalidAdquisitionModal  = errors.New("invalid adquisition modal")

	BaseURL    = "http://sicc.honducompras.gob.hn/HC/procesos/busquedahistorico.aspx"
	DetailsURL = "http://sicc.honducompras.gob.hn/HC/procesos/%s"

	adquisitionValues = map[string]AdquisitionValueType{
		"(Todas)":                            AdquisitionValueAllType,
		"Suministro de Bienes y/o Servicios": ServicesAndGoodsAdquisitionType,
		"Obras":                              WorksAdquisitionType,
		"Consultoria":                        ConsultingAdquisitionType,
	}

	adquisitionStatusValues = map[string]AdquisitionStatusType{
		"(Todas)":              AdquisitionStatusAllType,
		"Elaboración":          AdquisitionStatusElaborationType,
		"Revisado":             AdquisitionStatusReviewedType,
		"Recepción de Ofertas": AdquisitionStatusReceivedOffersType,
		"Evaluación":           AdquisitionStatusEvaluationType,
		"Adjudicado":           AdquisitionStatusAwardedType,
		"Desierto":             AdquisitionStatusDesertedType,
		"Fracasados":           AdquisitionStatusFailedType,
	}

	adquisitionModalValues = map[string]AdquisitionModalType{
		"(Todas)":               AdquisitionModalAllType,
		"Licitación o Concurso": AdquisitionModalLicitationOrContestType,
		"Compra Menor":          AdquisitionModalMinorPurchaseType,
		"Precalificación":       AdquisitionModalPrequalificationType,
	}
)

func NewFetcher(adquisitionType, adquisitionStatus, adquisitionModal string, client *client.Client, logger *slog.Logger) (*Fetcher, error) {
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
	adquisitionType   AdquisitionValueType
	adquisitionStatus AdquisitionStatusType
	adquisitionModal  AdquisitionModalType
	client            *client.Client
	logger            *slog.Logger
	pages             int64
	Report            *Report
}

func (f *Fetcher) Execute(ctx context.Context) error {
	page, err := f.doInintialRequest(ctx)
	if err != nil {
		return err
	}

	params, err := f.generateParams(page)
	if err != nil {
		return err
	}

	_, err = f.doSearchRequest(ctx, params)
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

func (f *Fetcher) doSearchRequest(ctx context.Context, params url.Values) (*webscraper.WebPage, error) {
	response, err := f.client.Post(ctx, BaseURL, nil, params)
	if err != nil {
		return nil, err
	}

	page := webscraper.NewWebPage(BaseURL, f.client, f.logger)

	if err := page.LoadFromResponse(ctx, response); err != nil {
		return nil, err
	}

	results, err := page.GetAttrs("#ctl00_cphCuerpo_gvResultados > tbody > tr > td:nth-child(5) > a", "href")
	if err != nil {
		return nil, err
	}

	strBuilder := strings.Builder{}

	for _, result := range results {
		strBuilder.WriteString(result)
		strBuilder.WriteString("\n")
	}

	html := strBuilder.String()

	err = os.WriteFile("page.txt", []byte(html), 0644)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (f *Fetcher) generateParams(page *webscraper.WebPage) (url.Values, error) {
	params := url.Values{}

	viewStateGenerator, err := page.GetAttr(ViewStateGeneratorAttr, "value")
	if err != nil {
		return nil, err
	}

	params.Set("__VIEWSTATEGENERATOR", viewStateGenerator)

	viewState, err := page.GetAttr(ViewStateAttr, "value")
	if err != nil {
		return nil, err
	}

	params.Set("__VIEWSTATE", viewState)

	eventValidation, err := page.GetAttr(EventValidationAttr, "value")
	if err != nil {
		return nil, err
	}

	params.Set("__EVENTVALIDATION", eventValidation)

	pages, err := page.GetTable("#ctl00_cphCuerpo_gvResultados > tbody > tr:nth-child(32) > td > table > tbody")
	if err != nil {
		return nil, err
	}

	fmt.Println(pages)

	// #ctl00_cphCuerpo_gvResultados > tbody > tr:nth-child(32) > td > table > tbody

	params.Set("ctl00_tkSM_HiddenField", "")
	params.Set("__EVENTTARGET", "")
	//params.Set("__EVENTTARGET", "ctl00$cphCuerpo$gvResultados")
	params.Set("__EVENTARGUMENT", "")
	//params.Set("__EVENTARGUMENT", "Page$2")
	params.Set("__LASTFOCUS", "")
	params.Set("ctl00$cphCuerpo$wpParametros_hidden", "")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlEntidades", "0")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlUC", "0")
	params.Set("ctl00_cphCuerpo_wpParametros_wdInicio_DrpPnl1_DP_CAL_ID_1", "%3Cx%20PostData%3D%222025x2x-1x-1x-1%22%3E%3C/x%3E")
	params.Set("ctl00$cphCuerpo$wpParametros$wdInicio_hidden", "")
	params.Set("ctl00_cphCuerpo_wpParametros_wdInicio_input", "(Todas)")
	params.Set("ctl00_cphCuerpo_wpParametros_wdFin_DrpPnl2_DP_CAL_ID_2", "%3Cx%20PostData%3D%222025x2x-1x-1x-1%22%3E%3C/x%3E")
	params.Set("ctl00_cphCuerpo_wpParametros_wdFin_input", "(Todas)")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlModalidad", "2")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlTipoAdquisicion", "1")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlEtapas", "0")
	params.Set("ctl00$cphCuerpo$wpParametros$ddlFteFinanciamiento", "0")
	params.Set("ctl00$cphCuerpo$wpParametros$txtObjetoCompra", "")
	params.Set("ctl00$cphCuerpo$wpParametros$txtExpediente", "")
	params.Set("ctl00$cphCuerpo$wpParametros$btnBuscar", "Buscar")
	params.Set("ctl00$_IG_CSS_LINKS_", "../StyleSheet.css|../ig_res/Default/ig_calendar.css|../ig_res/Default/ig_datechooser.css|../ig_res/Default/ig_panel.css|../ig_res/Default/ig_shared.css")

	return params, nil
}
