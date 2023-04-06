package customerio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Segment struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	State       string `json:"state,omitempty"`
	Type        string `json:"type,omitempty"`
}

func (c *APIClient) ListSegments(ctx context.Context) ([]Segment, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", "/v1/segments", nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: "/v1/segments", body: body}
	}

	var envelope struct {
		Segments []Segment `json:"segments"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	return envelope.Segments, nil
}

func (c *APIClient) GetSegment(ctx context.Context, id int) (Segment, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/segments/%d", id), nil)
	if err != nil {
		return Segment{}, err
	}
	if statusCode != http.StatusOK {
		return Segment{}, &CustomerIOError{status: statusCode, url: fmt.Sprintf("/v1/segments/%d", id), body: body}
	}

	var envelope struct {
		Segment Segment `json:"segment"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Segment{}, err
	}
	return envelope.Segment, nil
}
