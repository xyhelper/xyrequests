package xyrequests

import (
	"context"
	"errors"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

func TestDoWithRetry_NoRetry(t *testing.T) {
	c := &Client{}
	expectedErr := errors.New("boom")
	attempts := 0

	resp, err := c.doWithRetry(context.Background(), 0, func() (*http.Response, error) {
		attempts++
		return nil, expectedErr
	})

	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected err %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoWithRetry_SuccessAfterRetries(t *testing.T) {
	c := &Client{}
	expectedResp := &http.Response{StatusCode: 200}
	expectedErr := errors.New("temporary")
	attempts := 0

	resp, err := c.doWithRetry(context.Background(), 2, func() (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, expectedErr
		}
		return expectedResp, nil
	})

	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if resp != expectedResp {
		t.Fatalf("expected response pointer %#v, got %#v", expectedResp, resp)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestDoWithRetry_NegativeRetryFallsBackToOneAttempt(t *testing.T) {
	c := &Client{}
	expectedErr := errors.New("boom")
	attempts := 0

	_, err := c.doWithRetry(context.Background(), -3, func() (*http.Response, error) {
		attempts++
		return nil, expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected err %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt when maxRetry is negative, got %d", attempts)
	}
}
