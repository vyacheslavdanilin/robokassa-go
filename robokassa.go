package robokassa

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	CultureEn = "en"
	CultureRu = "ru"
)

var (
	ErrInvalidSum       = errors.New("invalid sum")
	ErrEmptyDescription = errors.New("empty description")
	ErrInvalidInvoiceId = errors.New("invalid invoice ID")
	ErrEmptyReceipt     = errors.New("empty receipt")
	ErrInvalidParam     = errors.New("invalid param")
)

type Payment struct {
	baseUrl              string
	baseInitRecurringUrl string
	baseRecurringUrl     string
	valid                bool
	data                 map[string]interface{}
	isTestMode           bool
	customParams         map[string]string
	login                string
	paymentPassword      string
	validationPassword   string
}

func NewPayment(login, paymentPassword, validationPassword string, testMode bool) *Payment {
	return &Payment{
		baseUrl:              "https://auth.robokassa.ru/Merchant/Index.aspx?",
		baseInitRecurringUrl: "https://auth.robokassa.ru/Merchant/Index.aspx?",
		baseRecurringUrl:     "https://auth.robokassa.ru/Merchant/Recurring",
		isTestMode:           testMode,
		data: map[string]interface{}{
			"MerchantLogin":  login,
			"InvId":          0,
			"OutSum":         0.0,
			"Desc":           "",
			"SignatureValue": "",
			"Encoding":       "utf-8",
			"Culture":        CultureRu,
			"IncCurrLabel":   "",
			"IsTest":         testMode,
			"Receipt":        nil,
		},
		customParams:       make(map[string]string),
		login:              login,
		paymentPassword:    paymentPassword,
		validationPassword: validationPassword,
	}
}

func (p *Payment) GetPaymentUrl(ctx context.Context, paymentType string) (string, error) {
	p.data["SignatureValue"] = p.getSignValue()

	data := url.Values{}
	for k, v := range p.data {
		data.Set(k, fmt.Sprintf("%v", v))
	}
	custom := url.Values{}
	for k, v := range p.customParams {
		custom.Set(k, v)
	}

	switch paymentType {
	case "base":
		return p.baseUrl + data.Encode() + "&" + custom.Encode(), nil
	case "init_recurring":
		return p.baseInitRecurringUrl + data.Encode() + "&" + custom.Encode(), nil
	case "recurring":
		return p.baseRecurringUrl, nil
	default:
		return "", ErrInvalidParam
	}
}

func (p *Payment) GetPaymentRecurringParams() (map[string]interface{}, error) {
	p.data["SignatureValue"] = p.getSignValue()
	return mergeMaps(p.data, convertMapStringToInterface(p.customParams)), nil
}

func (p *Payment) getSignValue() string {
	outSum, ok := p.data["OutSum"].(float64)
	if !ok || outSum <= 0 {
		panic(ErrInvalidSum)
	}

	desc, ok := p.data["Desc"].(string)
	if !ok || desc == "" {
		panic(ErrEmptyDescription)
	}

	invId, ok := p.data["InvId"].(int)
	if !ok || invId <= 0 {
		panic(ErrInvalidInvoiceId)
	}

	receipt, ok := p.data["Receipt"].(string)
	if !ok || receipt == "" {
		panic(ErrEmptyReceipt)
	}

	signature := fmt.Sprintf("%s:%0.2f:%d:%s:%s", p.login, outSum, invId, receipt, p.paymentPassword)
	if len(p.customParams) > 0 {
		keys := make([]string, 0, len(p.customParams))
		for k := range p.customParams {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			signature += fmt.Sprintf(":%s=%s", k, p.customParams[k])
		}
	}

	hash := md5.Sum([]byte(signature))
	return hex.EncodeToString(hash[:])
}

func (p *Payment) ValidateResult(data map[string]interface{}) bool {
	return p.validate(data, "validation")
}

func (p *Payment) ValidateSuccess(data map[string]interface{}) bool {
	return p.validate(data, "payment")
}

func (p *Payment) validate(data map[string]interface{}, passwordType string) bool {
	password := p.paymentPassword
	if passwordType == "validation" {
		password = p.validationPassword
	}

	signature := fmt.Sprintf("%v:%v:%s%s", data["OutSum"], data["InvId"], password, p.getCustomParamsString(data))
	hash := md5.Sum([]byte(signature))
	p.valid = strings.EqualFold(hex.EncodeToString(hash[:]), data["SignatureValue"].(string))

	return p.valid
}

func (p *Payment) IsValid() bool {
	return p.valid
}

func (p *Payment) AddCustomParameters(params map[string]string) error {
	if params == nil {
		return ErrInvalidParam
	}

	for k, v := range params {
		p.customParams["shp_"+k] = v
	}

	return nil
}

func (p *Payment) GetSuccessAnswer() string {
	return fmt.Sprintf("OK%d\n", p.getInvoiceId())
}

func (p *Payment) getCustomParamsString(source map[string]interface{}) string {
	params := make([]string, 0)

	for k, v := range source {
		if strings.HasPrefix(k, "shp_") {
			params = append(params, fmt.Sprintf("%s=%v", k, v))
		}
	}

	sort.Strings(params)
	if len(params) > 0 {
		return ":" + strings.Join(params, ":")
	}
	return ""
}

func (p *Payment) GetCustomParam(name string) interface{} {
	key := "shp_" + name
	if val, exists := p.data[key]; exists {
		return val
	}
	return nil
}

func (p *Payment) getInvoiceId() int {
	if invId, ok := p.data["InvId"].(int); ok {
		return invId
	}
	return 0
}

func (p *Payment) SetInvoiceId(id int) *Payment {
	p.data["InvId"] = id
	return p
}

func (p *Payment) SetPreviousInvoiceId(id int) *Payment {
	p.data["PreviousInvoiceID"] = id
	return p
}

func (p *Payment) GetSum() float64 {
	if sum, ok := p.data["OutSum"].(float64); ok {
		return sum
	}
	return 0
}

func (p *Payment) SetSum(sum float64) error {
	sum = float64(int(sum*100)) / 100
	if sum > 0 {
		p.data["OutSum"] = sum
		return nil
	}
	return ErrInvalidSum
}

func (p *Payment) GetDescription() string {
	if desc, ok := p.data["Desc"].(string); ok {
		return desc
	}
	return ""
}

func (p *Payment) SetDescription(description string) *Payment {
	p.data["Desc"] = description
	return p
}

func (p *Payment) GetCulture() string {
	if culture, ok := p.data["Culture"].(string); ok {
		return culture
	}
	return ""
}

func (p *Payment) SetCulture(culture string) *Payment {
	p.data["Culture"] = culture
	return p
}

func (p *Payment) GetCurrencyLabel() string {
	if label, ok := p.data["IncCurrLabel"].(string); ok {
		return label
	}
	return ""
}

func (p *Payment) SetCurrencyLabel(label string) *Payment {
	p.data["IncCurrLabel"] = label
	return p
}

func (p *Payment) SetEmail(email string) *Payment {
	p.data["Email"] = email
	return p
}

func (p *Payment) SetReceipt(receipt map[string]interface{}) *Payment {
	receiptJSON, _ := json.Marshal(receipt)
	p.data["Receipt"] = url.QueryEscape(string(receiptJSON))
	return p
}

func (p *Payment) SetRecurring() *Payment {
	p.data["Recurring"] = true
	return p
}

func mergeMaps(m1 map[string]interface{}, m2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m1)+len(m2))
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}

func convertMapStringToInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
