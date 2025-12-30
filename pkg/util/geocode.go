package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// KakaoGeocodeResponse represents the response from Kakao address search API
type KakaoGeocodeResponse struct {
	Documents []struct {
		Address struct {
			AddressName string `json:"address_name"`
			X           string `json:"x"` // longitude
			Y           string `json:"y"` // latitude
		} `json:"address"`
		RoadAddress struct {
			AddressName string `json:"address_name"`
			X           string `json:"x"` // longitude
			Y           string `json:"y"` // latitude
		} `json:"road_address"`
	} `json:"documents"`
	Meta struct {
		TotalCount int `json:"total_count"`
	} `json:"meta"`
}

// GeocodeAddress converts an address string to latitude and longitude using Kakao API
// Returns (latitude, longitude, error)
func GeocodeAddress(address string) (*float64, *float64, error) {
	if address == "" {
		return nil, nil, nil // No error, just no coordinates
	}

	kakaoAPIKey := os.Getenv("KAKAO_CLIENT_ID")
	if kakaoAPIKey == "" {
		return nil, nil, fmt.Errorf("KAKAO_CLIENT_ID not set in environment")
	}

	// Kakao Local API - Address Search
	// https://developers.kakao.com/docs/latest/ko/local/dev-guide#address-coord
	baseURL := "https://dapi.kakao.com/v2/local/search/address.json"

	// URL encode the address
	params := url.Values{}
	params.Add("query", address)
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Add("Authorization", fmt.Sprintf("KakaoAK %s", kakaoAPIKey))

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call Kakao API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("kakao API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var result KakaoGeocodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if any results found
	if result.Meta.TotalCount == 0 {
		return nil, nil, fmt.Errorf("no results found for address: %s", address)
	}

	// Get coordinates from the first result
	// Try road_address first, fall back to address
	var latStr, lngStr string

	if result.Documents[0].RoadAddress.Y != "" && result.Documents[0].RoadAddress.X != "" {
		latStr = result.Documents[0].RoadAddress.Y
		lngStr = result.Documents[0].RoadAddress.X
	} else if result.Documents[0].Address.Y != "" && result.Documents[0].Address.X != "" {
		latStr = result.Documents[0].Address.Y
		lngStr = result.Documents[0].Address.X
	} else {
		return nil, nil, fmt.Errorf("no coordinates in response")
	}

	// Parse latitude and longitude
	var lat, lng float64
	if _, err := fmt.Sscanf(latStr, "%f", &lat); err != nil {
		return nil, nil, fmt.Errorf("failed to parse latitude: %w", err)
	}
	if _, err := fmt.Sscanf(lngStr, "%f", &lng); err != nil {
		return nil, nil, fmt.Errorf("failed to parse longitude: %w", err)
	}

	return &lat, &lng, nil
}
