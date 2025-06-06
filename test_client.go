package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

func main() {
	// Проверяем health endpoint
	fmt.Println("Проверяем health endpoint...")
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Printf("Ошибка при обращении к health endpoint: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Ошибка чтения ответа: %v\n", err)
		return
	}

	fmt.Printf("Health check ответ (статус %d):\n%s\n\n", resp.StatusCode, string(body))

	// Если есть тестовое видео, отправляем его на анализ
	if len(os.Args) > 1 {
		videoPath := os.Args[1]
		fmt.Printf("Отправляем видео %s на анализ...\n", videoPath)
		
		err := testAnalyze(videoPath)
		if err != nil {
			fmt.Printf("Ошибка при тестировании анализа: %v\n", err)
		}
	} else {
		fmt.Println("Для тестирования анализа запустите: go run test_client.go <путь_к_видео>")
	}
}

func testAnalyze(videoPath string) error {
	// Читаем видео файл
	videoData, err := os.ReadFile(videoPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения видео файла: %w", err)
	}

	// Создаем multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавляем видео файл
	videoWriter, err := writer.CreateFormFile("video", "test_video.mp4")
	if err != nil {
		return fmt.Errorf("ошибка создания form field: %w", err)
	}
	
	if _, err := videoWriter.Write(videoData); err != nil {
		return fmt.Errorf("ошибка записи видео: %w", err)
	}

	// Добавляем координаты (пример для Москвы)
	writer.WriteField("startLat", "55.7558")
	writer.WriteField("startLon", "37.6176")
	writer.WriteField("endLat", "55.7568")
	writer.WriteField("endLon", "37.6186")
	writer.WriteField("segmentLength", "100")

	writer.Close()

	// Отправляем запрос
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/analyze", &body)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	fmt.Println("Отправляем запрос на анализ...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	fmt.Printf("Ответ анализа (статус %d):\n%s\n", resp.StatusCode, string(respBody))
	return nil
} 