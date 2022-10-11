package customerio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var ErrCustomerNotFound = errors.New("customer not found")

// Customer represents all of the fields we think of associated with a customer
// This includes cio_id which is not necessarily found in request/response
// bodies. That said--it's more of an entity definition than an api def (though
// we use it as both)
type Customer struct {
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	CioID      string                 `json:"cio_id,omitempty"`
	CreatedAt  *time.Time             `json:"created_at,omitempty"`
	Email      string                 `json:"email,omitempty"`
	ID         string                 `json:"id,omitempty"`
}

func (c Customer) MarshalJSON() ([]byte, error) {
	// Render non-nil times as epoch seconds
	type Alias Customer

	t := int64(0)
	if c.CreatedAt != nil && !c.CreatedAt.IsZero() {
		t = c.CreatedAt.UTC().Unix()
	}
	return json.Marshal(&struct {
		CreatedAt int64 `json:"created_at,omitempty"`
		Alias
	}{
		CreatedAt: t,
		Alias:     (Alias)(c),
	})
}

type searchResponse struct {
	Customer struct {
		Attributes struct {
			Attributes string `json:"attributes"`
			CioID      string `json:"cio_id"`
			CreatedAt  string `json:"created_at"`
			Email      string `json:"email"`
			ID         string `json:"id"`
		} `json:"attributes"`
	} `json:"customer"`
}

func (c *APIClient) CustomerSearch(ctx context.Context, id string, idType IdentifierType) (Customer, error) {
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
	resp := searchResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return Customer{}, err
	}

	attributes := map[string]interface{}{}
	if js, err := strconv.Unquote(resp.Customer.Attributes.Attributes); err != nil && js != "" {
		err = json.Unmarshal([]byte(js), &attributes)
		if err != nil {
			return Customer{}, err
		}
	}

	var thyme *time.Time
	if resp.Customer.Attributes.CreatedAt != "" {
		createdInt, err := strconv.Atoi(resp.Customer.Attributes.CreatedAt)
		if err != nil {
			return Customer{}, err
		}
		unixS := time.Unix(int64(createdInt), 0)
		thyme = &unixS
	}

	return Customer{
		Attributes: attributes,
		CioID:      resp.Customer.Attributes.CioID,
		CreatedAt:  thyme,
		Email:      resp.Customer.Attributes.Email,
		ID:         resp.Customer.Attributes.ID,
	}, nil
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
