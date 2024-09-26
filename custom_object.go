package customerio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type CustomObject struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	SingularName string `json:"singular_name"`
	Slug         string `json:"slug"`
	SingularSlug string `json:"singular_slug"`
}

type GetCustomObjectAttributesResponse struct {
	Object struct {
		Attributes map[string]any `json:"attributes"`
	} `json:"object" `
}

func (c *APIClient) ListCustomObjects(ctx context.Context) ([]CustomObject, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", "/v1/object_types", nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: "/v1/object_types", body: body}
	}

	var respObj struct {
		Types []CustomObject `json:"types"`
	}
	if err := json.Unmarshal(body, &respObj); err != nil {
		return nil, err
	}

	return respObj.Types, nil
}

func (c *APIClient) FindCustomObjects(ctx context.Context, objectTypeID string, filter map[string]any) ([]string, error) {
	body, statusCode, err := c.doRequest(ctx, "POST", "/v1/objects", map[string]any{
		"object_type_id": objectTypeID,
		"filter":         filter,
	})
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: "/v1/object_types", body: body}
	}

	var respObj struct {
		IDs []string `json:"ids"`
	}

	if err := json.Unmarshal(body, &respObj); err != nil {
		return nil, err
	}

	return respObj.IDs, nil
}

func (c *APIClient) GetCustomObjectAttributes(ctx context.Context, objectTypeID, objectID string) (map[string]any, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/objects/%s/%s/attributes", objectTypeID, objectID), nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: "/v1/object_types", body: body}
	}

	var respObj struct {
		Object struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"object" `
	}
	if err := json.Unmarshal(body, &respObj); err != nil {
		return nil, err
	}

	return respObj.Object.Attributes, nil
}

func (c *CustomerIO) TrackWriteBatch(ctx context.Context, actions []map[string]any) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("%s/api/v2/batch", c.URL), map[string]any{
		"batch": actions,
	})
	if err != nil {
		return err
	}

	return nil
}
