package collector

import (
	"time"

	"github.com/rdaniel1105/gob-vendor/client"
)

// AdquisitionType is the type for "tipo adquisición"
type AdquisitionType string

// AdquisitionStage is the type for "etapa de adquisición"
type AdquisitionStage string

// AdquisitionCategory is the type for "modalidad"
type AdquisitionCategory string

// CurrencyType is the type of currency
type CurrencyType string

const (
	AdquisitionTypeAll              AdquisitionType = "(Todas)"
	AdquisitionTypeServicesAndGoods AdquisitionType = "Suministro de Bienes y/o Servicios"
	AdquisitionTypeWorks            AdquisitionType = "Obras"
	AdquisitionTypeConsulting       AdquisitionType = "Consultoria"

	AdquisitionStageAll                AdquisitionStage = "(Todas)"
	AdquisitionStageElaborationType    AdquisitionStage = "Elaboración"
	AdquisitionStageReviewedType       AdquisitionStage = "Revisado"
	AdquisitionStageReceivedOffersType AdquisitionStage = "Recepción de Ofertas"
	AdquisitionStageEvaluationType     AdquisitionStage = "Evaluación"
	AdquisitionStageAwardedType        AdquisitionStage = "Adjudicado"
	AdquisitionStageDesertedType       AdquisitionStage = "Desierto"
	AdquisitionStageFailedType         AdquisitionStage = "Fracasados"

	AdquisitionCategoryAll                     AdquisitionCategory = "(Todas)"
	AdquisitionCategoryLicitationOrContestType AdquisitionCategory = "Licitación o Concurso"
	AdquisitionCategoryMinorPurchaseType       AdquisitionCategory = "Compra Menor"
	AdquisitionCategoryPrequalificationType    AdquisitionCategory = "Precalificación"
)

var (
	CurrencyHNL CurrencyType = "Lps."
	CurrencyUSD CurrencyType = "USD"
)

type ListItem struct {
	ExpedientID        string
	Entity             string
	Unit               string
	Object             string
	StartDate          string
	EndDate            string
	Link               string
	client             *client.Client
	viewState          string
	eventValidation    string
	viewStateGenerator string
}

type Detail struct {
	ExpedientID     string              `json:"expedient_id"`
	Entity          string              `json:"entity"`
	Unit            string              `json:"unit"`
	Object          string              `json:"object"`
	StartDate       time.Time           `json:"start_date"`
	ReceptionDate   time.Time           `json:"reception_date"`
	EndDate         time.Time           `json:"end_date"`
	SourceType      string              `json:"source_type"`
	Source          string              `json:"source"`
	Category        AdquisitionCategory `json:"category"`
	Stage           AdquisitionStage    `json:"stage"`
	AcquisitionType AdquisitionType     `json:"acquisition_type"`
	ReceptionPlace  string              `json:"reception_place"`
	Currency        CurrencyType        `json:"currency"`
	TenderCost      float64             `json:"tender_cost"`
	ContactName     string              `json:"contact_name"`
	ContactPhone    string              `json:"contact_phone"`
	ContactEmail    string              `json:"contact_email"`

	IDUNSPSC       string  `json:"id_unspec"`
	Description    string  `json:"description"`
	Specifications string  `json:"specifications"`
	Quantity       float64 `json:"quantity"`
}
