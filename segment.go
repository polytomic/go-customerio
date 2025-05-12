package customerio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Segment struct {
	ID            int    `json:"id,omitempty"`
	DeduplicateID string `json:"deduplicate_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	State         string `json:"state,omitempty"`
	Type          string `json:"type,omitempty"`
}

func (c *APIClient) CreateSegment(ctx context.Context, name, description string) (Segment, error) {
	body, statusCode, err := c.doRequest(ctx, "POST", "/v1/segments", map[string]any{
		"segment": map[string]any{
			"name":        name,
			"description": description,
		},
	})
	if err != nil {
		return Segment{}, fmt.Errorf("failed to create segment: %w", err)
	}

	if statusCode != http.StatusOK {
		return Segment{}, &CustomerIOError{status: statusCode, url: "/v1/segments", body: body}
	}

	var envelope struct {
		Segment Segment `json:"segment"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return Segment{}, fmt.Errorf("failed to unmarshal segment response: %w", err)
	}
	return envelope.Segment, nil
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

type Identifiers struct {
	CioID string `json:"cio_id"`
	ID    string `json:"id"`
	Email string `json:"email"`
}
type SegmentMemberResponse struct {
	IDs         []string      `json:"ids"`
	Identifiers []Identifiers `json:"identifiers"`
	Next        string        `json:"next"`
}

func (c *APIClient) ListSegmentMembers(ctx context.Context, segmentID int, start string) (*SegmentMemberResponse, error) {
	url := fmt.Sprintf("/v1/segments/%d/membership?limit=30000", segmentID)
	if start != "" {
		url = fmt.Sprintf("%s&start=%s", url, start)
	}
	body, statusCode, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, &CustomerIOError{status: statusCode, url: fmt.Sprintf("/segments/%d/membership", segmentID), body: body}
	}

	var respObj SegmentMemberResponse
	if err := json.Unmarshal(body, &respObj); err != nil {
		return nil, err
	}
	return &respObj, nil
}
