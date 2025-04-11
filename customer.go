package customerio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrCustomerNotFound = errors.New("customer not found")

// Customer represents all of the fields we think of associated with a customer
// This includes cio_id which is not necessarily found in request/response
// bodies. That said--it's more of an entity definition than an api def (though
// we use it as both)
type Customer struct {
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	CioID        string                 `json:"cio_id,omitempty"`
	CreatedAt    *time.Time             `json:"created_at,omitempty"`
	Email        string                 `json:"email,omitempty"`
	ID           string                 `json:"id,omitempty"`
	Unsubscribed *bool                  `json:"unsubscribed,omitempty"`
}

/*
	"customer": {
			"id": "",
			"identifiers": {
				"cio_id": "afee09000001",
				"email": "nia.kunde@feedstock.com"
			},
			"attributes": {
				"cio_id": "afee09000001",
				"email": "nia.kunde@feedstock.com",
				"name": "{\"first_name\":\"Nia\",\"last_name\":\"Kunde\"}"
			},
			"timestamps": {
				"cio_id": 1715780449,
				"email": 0,
				"name": 1715780447
			},
			"unsubscribed": false,
			"devices": []
		}
*/
type attributesResponse struct {
	Customer struct {
		ID          string `json:"id"`
		Identifiers struct {
			CioID string `json:"cio_id"`
			Email string `json:"email"`
		} `json:"identifiers"`
		Attributes   map[string]any   `json:"attributes"`
		Timestamps   map[string]int64 `json:"timestamps"`
		Unsubscribed bool             `json:"unsubscribed"`
	} `json:"customer"`
}

type customerioRelationshipResponse struct {
	CioRelationships []RelationshipsResponse `json:"cio_relationships"`
	Next             string
}

func (c *APIClient) GetCustomerRelationships(ctx context.Context, id string, idType IdentifierType) ([]RelationshipsResponse, error) {
	v := url.Values{}
	v.Add("id_type", string(idType))
	return c.getRelationships(ctx, fmt.Sprintf("/v1/customers/%s/relationships", id), v)
}

func (c *APIClient) getRelationships(ctx context.Context, rootURL string, defualtValues url.Values) ([]RelationshipsResponse, error) {
	var rels []RelationshipsResponse
	start := ""
	for {
		v := url.Values{}
		for k, vList := range defualtValues {
			for _, val := range vList {
				v.Add(k, val)
			}
		}

		v.Add("limit", "100")
		if start != "" {
			v.Add("start", start)
		}
		qs := v.Encode()
		url := fmt.Sprintf("%s?%s", rootURL, qs)
		body, statusCode, err := c.doRequest(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		if statusCode != http.StatusOK {
			return nil, &CustomerIOError{status: statusCode, url: url, body: body}
		}

		var resp customerioRelationshipResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}

		rels = append(rels, resp.CioRelationships...)
		if resp.Next == "" {
			break
		}

		start = resp.Next
	}

	return rels, nil
}

type FindCustomersIdentity struct {
	CioID string `json:"cio_id"`
	ID    string `json:"id"`
	Email string `json:"email"`
}

type FindCustomersResponse struct {
	Identifiers []FindCustomersIdentity `json:"identifiers"`
	Next        string                  `json:"next"`
}

func (c *APIClient) FindCustomers(ctx context.Context, filter map[string]any, start string) (FindCustomersResponse, error) {
	url := "/v1/customers?limit=1000"
	if start != "" {
		url = fmt.Sprintf("%s&start=%s", url, start)
	}

	body, statusCode, err := c.doRequest(ctx, "POST", url, map[string]any{
		"filter": filter,
	})
	if err != nil {
		return FindCustomersResponse{}, err
	}
	if statusCode != http.StatusOK {
		return FindCustomersResponse{}, &CustomerIOError{status: statusCode, url: url, body: body}
	}

	var respObj FindCustomersResponse

	if err := json.Unmarshal(body, &respObj); err != nil {
		return FindCustomersResponse{}, err
	}

	return respObj, nil
}

func (c *APIClient) GetCustomer(ctx context.Context, id string, idType IdentifierType) (Customer, error) {
	v := url.Values{}
	v.Add("id_type", string(idType))
	qs := v.Encode()
	url := fmt.Sprintf("/v1/customers/%s/attributes?%s", id, qs)
	body, statusCode, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return Customer{}, err
	}

	if statusCode == http.StatusNotFound {
		return Customer{}, ErrCustomerNotFound
	} else if statusCode != http.StatusOK {
		return Customer{}, &CustomerIOError{status: statusCode, url: url, body: body}
	}
	resp := attributesResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return Customer{}, err
	}

	cust := Customer{
		Attributes:   resp.Customer.Attributes,
		CioID:        resp.Customer.Identifiers.CioID,
		Email:        resp.Customer.Identifiers.Email,
		ID:           resp.Customer.ID,
		Unsubscribed: &resp.Customer.Unsubscribed,
	}

	if ts, ok := resp.Customer.Timestamps["cio_id"]; ok {
		unixS := time.Unix(int64(ts), 0)
		cust.CreatedAt = &unixS
	}

	return cust, nil
}

type customerSearchRequest struct {
	Filter filterCondition `json:"filter"`
}
type filterCondition struct {
	Or  []attributeCondition `json:"or,omitempty"`
	And []attributeCondition `json:"and,omitempty"`
}
type attributeCondition struct {
	Attribute attribute `json:"attribute"`
}

type attribute struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type searchResponse struct {
	Identifiers []map[string]interface{} `json:"identifiers"`
}

// NewEqAttribute takes a field and string and produces an Equality
// AttributeCondition
func NewEqAttribute(field string, value string) attributeCondition {
	return attributeCondition{
		Attribute: attribute{
			Field:    field,
			Operator: "eq",
			Value:    value,
		},
	}
}

// LookupCustomerioIds takes a list of emails/ids/cio ids and returns a list of
// the same size with the valid (if any) cio ids.
func (c *APIClient) LookupCustomerioIds(ctx context.Context, ids []string, idType IdentifierType) ([]string, error) {
	// A better thing to do would be to split these into batches and then issue
	// requests, one per 1k results. This is just a nicety at this point, so
	// I'll leave that for another time.
	if len(ids) > 1000 {
		return nil, errors.New("Can only lookup 1k customers at a time")
	}
	conditions := make([]attributeCondition, len(ids))
	for i, id := range ids {
		conditions[i] = NewEqAttribute(string(idType), id)
	}
	payload := customerSearchRequest{
		Filter: filterCondition{Or: conditions},
	}
	url := "/v1/customers?limit=1000"
	body, statusCode, err := c.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: url, body: body}
	}
	resp := searchResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	lookup := map[string]string{}
	for _, result := range resp.Identifiers {
		lookup[fmt.Sprint(result[string(idType)])] = fmt.Sprint(result["cio_id"])
	}

	result := make([]string, len(ids))
	for i, id := range ids {
		if idType == IdentifierTypeEmail {
			id = strings.ToLower(id)
		}
		result[i] = lookup[id]
	}
	return result, nil
}

type emailSearchResponse struct {
	Results []struct {
		CioID string `json:"cio_id"`
	} `json:"results"`
}

func (c *APIClient) LookupCustomersByEmail(ctx context.Context, email string) ([]string, error) {
	v := url.Values{}
	v.Add("email", string(email))
	qs := v.Encode()
	url := fmt.Sprintf("/v1/customers?%s", qs)
	body, statusCode, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, ErrCustomerNotFound
	} else if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: url, body: body}
	}
	resp := emailSearchResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	cioids := make([]string, len(resp.Results))
	for i, r := range resp.Results {
		cioids[i] = r.CioID
	}
	return cioids, nil
}
