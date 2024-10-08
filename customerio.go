package customerio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const DefaultUserAgent = "Customer.io Go Client/" + Version

// CustomerIO wraps the customer.io track API, see: https://customer.io/docs/api/#apitrackintroduction
type CustomerIO struct {
	siteID    string
	apiKey    string
	URL       string
	UserAgent string
	Client    *http.Client
}

// CustomerIOError is returned by any method that fails at the API level
type CustomerIOError struct {
	status int
	url    string
	body   []byte
}

func (e *CustomerIOError) Error() string {
	return fmt.Sprintf("%v: %v %v", e.status, e.url, string(e.body))
}

// ParamError is an error returned if a parameter to the track API is invalid.
type ParamError struct {
	Param string // Param is the name of the parameter.
}

func (e ParamError) Error() string { return e.Param + ": missing" }

// NewTrackClient prepares a client for use with the Customer.io track API, see: https://customer.io/docs/api/#apitrackintroduction
// using a Tracking Site ID and API Key pair from https://fly.customer.io/settings/api_credentials
func NewTrackClient(siteID, apiKey string, opts ...option) *CustomerIO {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}
	c := &CustomerIO{
		siteID:    siteID,
		apiKey:    apiKey,
		URL:       "https://track.customer.io",
		UserAgent: DefaultUserAgent,
		Client:    client,
	}

	for _, opt := range opts {
		opt.track(c)
	}

	return c
}

// NewCustomerIO prepares a client for use with the Customer.io track API, see: https://customer.io/docs/api/#apitrackintroduction
// deprecated in favour of NewTrackClient
func NewCustomerIO(siteID, apiKey string) *CustomerIO {
	return NewTrackClient(siteID, apiKey)
}

// IdentifyCtx identifies a customer and sets their attributes
func (c *CustomerIO) IdentifyCtx(ctx context.Context, customerID string, attributes map[string]interface{}) error {
	if customerID == "" {
		return ParamError{Param: "customerID"}
	}
	_, err := c.request(ctx, "PUT",
		fmt.Sprintf("%s/api/v1/customers/%s", c.URL, url.PathEscape(customerID)),
		attributes)
	return err
}

// Identify identifies a customer and sets their attributes
func (c *CustomerIO) Identify(customerID string, attributes map[string]interface{}) error {
	return c.IdentifyCtx(context.Background(), customerID, attributes)
}

// TrackCtx sends a single event to Customer.io for the supplied user
func (c *CustomerIO) TrackCtx(ctx context.Context, customerID string, eventName string, data map[string]interface{}) error {
	if customerID == "" {
		return ParamError{Param: "customerID"}
	}
	if eventName == "" {
		return ParamError{Param: "eventName"}
	}
	_, err := c.request(ctx, "POST",
		fmt.Sprintf("%s/api/v1/customers/%s/events", c.URL, url.PathEscape(customerID)),
		map[string]interface{}{
			"name": eventName,
			"data": data,
		})
	return err
}

// Track sends a single event to Customer.io for the supplied user
func (c *CustomerIO) Track(customerID string, eventName string, data map[string]interface{}) error {
	return c.TrackCtx(context.Background(), customerID, eventName, data)
}

// TrackAnonymousCtx sends a single event to Customer.io for the anonymous user
func (c *CustomerIO) TrackAnonymousCtx(ctx context.Context, anonymousID, eventName string, data map[string]interface{}) error {
	if eventName == "" {
		return ParamError{Param: "eventName"}
	}

	payload := map[string]interface{}{
		"name": eventName,
		"data": data,
	}

	if anonymousID != "" {
		payload["anonymous_id"] = anonymousID
	}

	_, err := c.request(ctx, "POST", fmt.Sprintf("%s/api/v1/events", c.URL), payload)
	return err
}

// TrackAnonymous sends a single event to Customer.io for the anonymous user
func (c *CustomerIO) TrackAnonymous(anonymousID, eventName string, data map[string]interface{}) error {
	return c.TrackAnonymousCtx(context.Background(), anonymousID, eventName, data)
}

// DeleteCtx deletes a customer
func (c *CustomerIO) DeleteCtx(ctx context.Context, customerID string) error {
	if customerID == "" {
		return ParamError{Param: "customerID"}
	}
	_, err := c.request(ctx, "DELETE",
		fmt.Sprintf("%s/api/v1/customers/%s", c.URL, url.PathEscape(customerID)),
		nil)
	return err
}

// Delete deletes a customer
func (c *CustomerIO) Delete(customerID string) error {
	return c.DeleteCtx(context.Background(), customerID)
}

// AddDeviceCtx adds a device for a customer
func (c *CustomerIO) AddDeviceCtx(ctx context.Context, customerID string, deviceID string, platform string, data map[string]interface{}) error {
	if customerID == "" {
		return ParamError{Param: "customerID"}
	}
	if deviceID == "" {
		return ParamError{Param: "deviceID"}
	}
	if platform == "" {
		return ParamError{Param: "platform"}
	}

	body := map[string]map[string]interface{}{
		"device": {
			"id":       deviceID,
			"platform": platform,
		},
	}
	for k, v := range data {
		body["device"][k] = v
	}
	_, err := c.request(ctx, "PUT",
		fmt.Sprintf("%s/api/v1/customers/%s/devices", c.URL, url.PathEscape(customerID)),
		body)
	return err
}

// AddDevice adds a device for a customer
func (c *CustomerIO) AddDevice(customerID string, deviceID string, platform string, data map[string]interface{}) error {
	return c.AddDeviceCtx(context.Background(), customerID, deviceID, platform, data)
}

// DeleteDeviceCtx deletes a device for a customer
func (c *CustomerIO) DeleteDeviceCtx(ctx context.Context, customerID string, deviceID string) error {
	if customerID == "" {
		return ParamError{Param: "customerID"}
	}
	if deviceID == "" {
		return ParamError{Param: "deviceID"}
	}
	_, err := c.request(ctx, "DELETE",
		fmt.Sprintf("%s/api/v1/customers/%s/devices/%s", c.URL, url.PathEscape(customerID), url.PathEscape(deviceID)),
		nil)
	return err
}

// DeleteDevice deletes a device for a customer
func (c *CustomerIO) DeleteDevice(customerID string, deviceID string) error {
	return c.DeleteDeviceCtx(context.Background(), customerID, deviceID)
}

func (c *CustomerIO) auth() string {
	return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", c.siteID, c.apiKey)))
}

func (c *CustomerIO) request(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var req *http.Request
	if body != nil {
		j, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(j))
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)

		req.Header.Add("User-Agent", c.UserAgent)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Content-Length", strconv.Itoa(len(j)))
	} else {
		var err error
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	req.Header.Add("Authorization", fmt.Sprintf("Basic %v", c.auth()))

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &CustomerIOError{
			status: resp.StatusCode,
			url:    url,
			body:   responseBody,
		}
	}

	return responseBody, nil
}

type IdentifierType string

const (
	IdentifierTypeID    IdentifierType = "id"
	IdentifierTypeEmail IdentifierType = "email"
	IdentifierTypeCioID IdentifierType = "cio_id"

	IdentifierTypeName        IdentifierType = "name"
	IdentifierTypeCioObjectID IdentifierType = "cio_object_id"
	IdentifierTypeObjectID    IdentifierType = "object_id"
)

type Identifier struct {
	Type  IdentifierType
	Value string
}

func (id Identifier) kv() map[string]string {
	return map[string]string{
		string(id.Type): id.Value,
	}
}

func (id Identifier) validate() error {
	if !(id.Type == IdentifierTypeID ||
		id.Type == IdentifierTypeEmail ||
		id.Type == IdentifierTypeCioID) {
		return errors.New("invalid id type")
	}

	if strings.TrimSpace(id.Value) == "" {
		return errors.New("invalid id")
	}

	return nil
}

// MergeCustomersCtx sends a request to Customer.io to merge two customer profiles together.
func (c *CustomerIO) MergeCustomersCtx(ctx context.Context, primary Identifier, secondary Identifier) error {
	if primary.validate() != nil {
		return ParamError{Param: "primary"}
	}
	if secondary.validate() != nil {
		return ParamError{Param: "secondary"}
	}

	_, err := c.request(ctx, "POST",
		fmt.Sprintf("%s/api/v1/merge_customers", c.URL),
		map[string]interface{}{
			"primary":   primary.kv(),
			"secondary": secondary.kv(),
		})
	return err
}

type RegionResponse struct {
	Url           string `json:"url"`
	Region        string `json:"region"`
	EnvironmentId int    `json:"environment_id"`
}

func (c *CustomerIO) Region(ctx context.Context) (RegionResponse, error) {
	body, err := c.request(ctx, "GET",
		fmt.Sprintf("%s/api/v1/accounts/region", c.URL),
		nil,
	)
	if err != nil {
		return RegionResponse{}, err
	}
	r := RegionResponse{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return RegionResponse{}, err
	}
	return r, nil

}

// MergeCustomers sends a request to Customer.io to merge two customer profiles together.
func (c *CustomerIO) MergeCustomers(primary Identifier, secondary Identifier) error {
	return c.MergeCustomersCtx(context.Background(), primary, secondary)
}

func (c *CustomerIO) AddOrUpdate(ctx context.Context, id string, req *Customer) error {
	outgoingAtts := map[string]interface{}{}
	for k, v := range req.Attributes {
		outgoingAtts[k] = v
	}
	if req.CreatedAt != nil {
		outgoingAtts["created_at"] = req.CreatedAt.UTC().Unix()
	}
	if req.Email != "" {
		outgoingAtts["email"] = req.Email
	}
	if req.ID != "" {
		outgoingAtts["id"] = req.ID
	}
	if req.Unsubscribed != nil {
		outgoingAtts["unsubscribed"] = req.Unsubscribed
	}

	url := fmt.Sprintf("%s/api/v1/customers/%s", c.URL, id)
	_, err := c.request(ctx, "PUT", url, outgoingAtts)
	if err != nil {
		return err
	}

	return nil
}

// AddCustomersToSegment adds customers to an existing manual segment. The
// customers will be identified by the specified identifier type. Customers
// without a value for that identifier will be skipped. The first return value
// is the number of identities that we attempted to add to the segment.
func (c *CustomerIO) AddCustomersToSegment(ctx context.Context, segmentID int, customers []Customer, identifier IdentifierType) (int, error) {
	identifiers := make([]string, 0, len(customers))
	for _, customer := range customers {
		switch identifier {
		case IdentifierTypeID:
			identifiers = append(identifiers, customer.ID)
		case IdentifierTypeEmail:
			identifiers = append(identifiers, customer.Email)
		case IdentifierTypeCioID:
			identifiers = append(identifiers, customer.CioID)
		}
	}

	_, err := c.request(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/segments/%d/add_customers?id_type=%s", c.URL, segmentID, identifier),
		map[string]interface{}{
			"ids": identifiers,
		},
	)
	return len(identifiers), err
}
