package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"road-detector-go/pkg/models"
	"github.com/sirupsen/logrus"
)

// PythonAPIClient клиент для взаимодействия с Python FastAPI
type PythonAPIClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewPythonAPIClient создает новый клиент для Python API
func NewPythonAPIClient(baseURL string, timeout time.Duration, logger *logrus.Logger) *PythonAPIClient {
	return &PythonAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// AnalyzeVideo отправляет видео на анализ в Python API
func (c *PythonAPIClient) AnalyzeVideo(request models.AnalyzeRequest) (*models.PythonAPIResponse, error) {
	c.logger.Info("Отправка запроса на анализ видео в Python API")

	// Создаем multipart form-data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавляем видео файл
	videoWriter, err := writer.CreateFormFile("video", request.VideoFilename)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания form field для видео: %w", err)
	}
	
	if _, err := videoWriter.Write(request.VideoData); err != nil {
		return nil, fmt.Errorf("ошибка записи видео данных: %w", err)
	}

	// Добавляем startLat
	if err := writer.WriteField("startLat", fmt.Sprintf("%.6f", request.StartPoint.Lat)); err != nil {
		return nil, fmt.Errorf("ошибка записи startLat: %w", err)
	}

	// Добавляем startLon
	if err := writer.WriteField("startLon", fmt.Sprintf("%.6f", request.StartPoint.Lon)); err != nil {
		return nil, fmt.Errorf("ошибка записи startLon: %w", err)
	}

	// Добавляем endLat
	if err := writer.WriteField("endLat", fmt.Sprintf("%.6f", request.EndPoint.Lat)); err != nil {
		return nil, fmt.Errorf("ошибка записи endLat: %w", err)
	}

	// Добавляем endLon
	if err := writer.WriteField("endLon", fmt.Sprintf("%.6f", request.EndPoint.Lon)); err != nil {
		return nil, fmt.Errorf("ошибка записи endLon: %w", err)
	}

	// Добавляем segmentLength
	if err := writer.WriteField("segmentLength", fmt.Sprintf("%d", request.SegmentLength)); err != nil {
		return nil, fmt.Errorf("ошибка записи segmentLength: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("ошибка закрытия multipart writer: %w", err)
	}

	// Создаем HTTP запрос
	url := fmt.Sprintf("%s/analyze", c.baseURL)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Отправляем запрос
	c.logger.Debugf("Отправка POST запроса на %s", url)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Python API вернул ошибку: статус %d, тело: %s", resp.StatusCode, string(respBody))
	}

	// Парсим JSON ответ
	var apiResponse models.PythonAPIResponse
	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON ответа: %w", err)
	}

	c.logger.Info("Успешно получен ответ от Python API")
	return &apiResponse, nil
}

// CheckHealth проверяет состояние Python API
func (c *PythonAPIClient) CheckHealth() (*models.HealthResponse, error) {
	c.logger.Debug("Проверка здоровья Python API")

	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Python API вернул ошибку: статус %d, тело: %s", resp.StatusCode, string(respBody))
	}

	var healthResponse models.HealthResponse
	if err := json.Unmarshal(respBody, &healthResponse); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON ответа: %w", err)
	}

	return &healthResponse, nil
} 