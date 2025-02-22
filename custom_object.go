package customerio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	OperatorExists string = "exists"
	OperatorEq     string = "eq"
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

type RelationshipsResponse struct {
	ObjectTypeID string `json:"object_type_id"`
	Identifiers  struct {
		CioID       string `json:"cio_id"`
		Email       string `json:"email"`
		ID          string `json:"id"`
		ObjectID    string `json:"object_id"`
		CioObjectID string `json:"cio_object_id"`
	} `json:"identifiers"`
	Attributes map[string]any `json:"attributes"`
	Timestamps map[string]any `json:"timestamps"`
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

type ObjectAttribute struct {
	TypeID   string `json:"type_id"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type ObjectAttributeCondition struct {
	Attribute ObjectAttribute `json:"object_attribute,omitempty"`
}

type CustomObjectFilter struct {
	Attribute *ObjectAttribute           `json:"object_attribute,omitempty"`
	Or        []ObjectAttributeCondition `json:"or,omitempty"`
	And       []ObjectAttributeCondition `json:"and,omitempty"`
	Not       *ObjectAttributeCondition  `json:"not,omitempty"`
}

func (c *APIClient) FindCustomObjects(ctx context.Context, objectTypeID string, filter CustomObjectFilter) ([]string, error) {
	body, statusCode, err := c.doRequest(ctx, "POST", "/v1/objects", map[string]any{
		"object_type_id": objectTypeID,
		"filter":         filter,
	})
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: "/v1/objects", body: body}
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
	url := fmt.Sprintf("/v1/objects/%s/%s/attributes", objectTypeID, objectID)
	body, statusCode, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: url, body: body}
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

func (c *APIClient) GetCustomObjectRelationships(ctx context.Context, objectTypeID, objectID string) ([]RelationshipsResponse, error) {
	return c.getRelationships(ctx, fmt.Sprintf("/v1/objects/%s/%s/relationships", objectTypeID, objectID), nil)
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
