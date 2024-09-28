package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

var (
	errNilBody    = errors.New("no response body")
	errStatusCode = errors.New("not OK status code")
)

// SlogErrorAttr just a simple helper function to return error attr for slog logger
func SlogErrorAttr(err error) slog.Attr {
	return slog.Any("err", err)
}

func SlogUrlAttr(req *http.Request) slog.Attr {
	if req == nil || req.URL == nil {
		return slog.String("url", "")
	}
	return slog.String("url", req.URL.String())
}

func HandleHTTPRequest(req *http.Request, httpClient http.Client) ([]byte, error) {
	response, err := httpClient.Do(req)
	if err != nil {
		slog.Error("request do", SlogUrlAttr(req), SlogErrorAttr(err))
		return nil, err
	}

	if response.Body == nil {
		slog.Error("request", SlogUrlAttr(req), SlogErrorAttr(errNilBody))
		return nil, errNilBody
	}
	defer response.Body.Close()

	jsonResp, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("response read", SlogUrlAttr(req), SlogErrorAttr(err))
		return nil, err
	}

	if response.StatusCode >= 400 {
		resp := make(map[string]any)
		err = json.Unmarshal(jsonResp, &resp)
		if err != nil {
			slog.Error("response read", SlogUrlAttr(req), SlogErrorAttr(err))
			return nil, fmt.Errorf("%w: %w, %d", err, errStatusCode, response.StatusCode)
		}
		slog.Error("response", slog.Any("response-body", resp))
		return nil, fmt.Errorf("%w, %d", errStatusCode, response.StatusCode)
	}

	return jsonResp, nil
}
