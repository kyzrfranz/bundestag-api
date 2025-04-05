package img

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"time"
)

// EnsureImage checks if the image is already cached and fresh.
func EnsureImage(res any, id string) error {
	cachePath := fmt.Sprintf(".img/%s.webp", id)

	// Check if cached file exists and is fresh
	if fi, err := os.Stat(cachePath); err == nil {
		if time.Since(fi.ModTime()) < 30*24*time.Hour { // within 1 month
			return nil
		}
	}

	// Get URL from res.PhotoLargeURL using reflection
	val := reflect.ValueOf(res)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	field := val.FieldByName("PhotoLargeURL")
	if !field.IsValid() || field.Kind() != reflect.String {
		return fmt.Errorf("PhotoLargeURL not found or invalid")
	}
	photoURL := field.String()

	// Fetch image
	resp, err := http.Get(photoURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	// Ensure .img directory exists
	if err := os.MkdirAll(".img", 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	tmpPath := fmt.Sprintf(".img/%s-tmp.jpg", id)
	dstPath := fmt.Sprintf(".img/%s.webp", id)

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp image file: %w", err)
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write image: %w", err)
	}
	tmpFile.Close()

	if err := convertToWebP(tmpPath, dstPath); err != nil {
		return fmt.Errorf("failed to convert image to WebP: %w", err)
	}
	os.Remove(tmpPath)
	return err
}

func convertToWebP(srcPath, dstPath string) error {
	cmd := exec.Command("cwebp", "-quiet", srcPath, "-o", dstPath)
	return cmd.Run()
}
