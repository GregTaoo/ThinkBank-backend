package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// ModelService 抽象接口
type ModelService interface {
	AnalyzeImage(path string) (caption string, embedding []float32, err error)
	AnalyzeText(text string) (embedding []float32, err error)
}

// HTTPModelService 调用服务
type HTTPModelService struct {
	URL string
}

func NewHTTPModelService(URL string) *HTTPModelService {
	return &HTTPModelService{URL: URL}
}

func (s *HTTPModelService) AnalyzeImage(path string) (string, []float32, error) {
	form := url.Values{}
	form.Set("url", path)

	reqURL := fmt.Sprintf("%s/analyzeImage", s.URL)
	resp, err := http.PostForm(reqURL, form)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Println("Failed to close response body:", e)
		}
	}()

	if resp.StatusCode != 200 {
		return "", nil, fmt.Errorf("model service error: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Caption   string    `json:"caption"`
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil, err
	}

	return result.Caption, result.Embedding, nil
}

func (s *HTTPModelService) AnalyzeText(text string) ([]float32, error) {
	form := url.Values{}
	form.Set("text", text)

	reqURL := fmt.Sprintf("%s/analyzeText", s.URL)
	resp, err := http.PostForm(reqURL, form)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Println("Failed to close response body:", e)
		}
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("model service error: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Embedding, nil
}
