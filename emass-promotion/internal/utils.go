package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"time"
)

func includesInt64(a []int64, s int64) bool {
	for _, value := range a {
		if value == s {
			return true
		}
	}

	return false
}

func isInDayRange(t time.Time, days int) bool {
	return time.Since(t) < time.Duration(days*24)*time.Hour
}

func gZip(data []byte) (string, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		return "", fmt.Errorf("failed to write data: %v", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip: %v", err)
	}
	gzippedData := b.Bytes()
	encodedData := base64.StdEncoding.EncodeToString(gzippedData)

	return encodedData, nil
}
