package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stakwork/sphinx-tribes/auth"
	"github.com/stakwork/sphinx-tribes/db"
	mocks "github.com/stakwork/sphinx-tribes/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTribesByOwner(t *testing.T) {
	mockDb := mocks.NewDatabase(t)
	tHandler := NewTribeHandler(mockDb)

	t.Run("Should test that all tribes that an owner did not delete are returned if all=true is added to the request query", func(t *testing.T) {
		// Mock data
		mockPubkey := "mock_pubkey"
		mockTribes := []db.Tribe{
			{UUID: "uuid", OwnerPubKey: mockPubkey, Deleted: false},
			{UUID: "uuid", OwnerPubKey: mockPubkey, Deleted: false},
		}
		mockDb.On("GetAllTribesByOwner", mock.Anything).Return(mockTribes).Once()

		// Create request with "all=true" query parameter
		req, err := http.NewRequest("GET", "/tribes_by_owner/"+mockPubkey+"?all=true", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.GetTribesByOwner)
		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData []db.Tribe
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("Error decoding JSON response: %s", err)
		}
		assert.ElementsMatch(t, mockTribes, responseData)
	})

	t.Run("Should test that all tribes that are not unlisted by an owner are returned", func(t *testing.T) {
		// Mock data
		mockPubkey := "mock_pubkey"
		mockTribes := []db.Tribe{
			{UUID: "uuid", OwnerPubKey: mockPubkey, Unlisted: false},
			{UUID: "uuid", OwnerPubKey: mockPubkey, Unlisted: false},
		}
		mockDb.On("GetTribesByOwner", mock.Anything).Return(mockTribes)

		// Create request without "all=true" query parameter
		req, err := http.NewRequest("GET", "/tribes/"+mockPubkey, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.GetTribesByOwner)
		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData []db.Tribe
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("Error decoding JSON response: %s", err)
		}
		assert.ElementsMatch(t, mockTribes, responseData)
	})
}

func TestGetTribe(t *testing.T) {
	mockDb := mocks.NewDatabase(t)
	tHandler := NewTribeHandler(mockDb)

	t.Run("Should test that a tribe can be returned when the right UUID is passed to the request parameter", func(t *testing.T) {
		// Mock data
		mockUUID := "valid_uuid"
		mockTribe := db.Tribe{
			UUID: mockUUID,
		}
		mockChannels := []db.Channel{
			{ID: 1, TribeUUID: mockUUID},
			{ID: 2, TribeUUID: mockUUID},
		}
		mockDb.On("GetTribe", mock.Anything).Return(mockTribe).Once()
		mockDb.On("GetChannelsByTribe", mock.Anything).Return(mockChannels).Once()

		// Serve request
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("uuid", mockUUID)
		req, err := http.NewRequestWithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx), http.MethodGet, "/"+mockUUID, nil)
		if err != nil {
			t.Fatal(err)
		}

		handler := http.HandlerFunc(tHandler.GetTribe)
		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("Error decoding JSON response: %s", err)
		}
		assert.Equal(t, mockTribe.UUID, responseData["uuid"])
	})

	t.Run("Should test that no tribe is returned when a nonexistent UUID is passed", func(t *testing.T) {
		// Mock data
		mockDb.ExpectedCalls = nil
		nonexistentUUID := "nonexistent_uuid"
		mockDb.On("GetTribe", nonexistentUUID).Return(db.Tribe{}).Once()
		mockDb.On("GetChannelsByTribe", mock.Anything).Return([]db.Channel{}).Once()

		// Serve request
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("uuid", nonexistentUUID)
		req, err := http.NewRequestWithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx), http.MethodGet, "/"+nonexistentUUID, nil)
		if err != nil {
			t.Fatal(err)
		}

		handler := http.HandlerFunc(tHandler.GetTribe)
		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("Error decoding JSON response: %s", err)
		}
		assert.Equal(t, "", responseData["uuid"])
	})
}

func TestGetTribesByAppUrl(t *testing.T) {
	mockDb := mocks.NewDatabase(t)
	tHandler := NewTribeHandler(mockDb)

	t.Run("Should test that a tribe is returned when the right app URL is passed", func(t *testing.T) {
		// Mock data
		mockAppURL := "valid_app_url"
		mockTribes := []db.Tribe{
			{UUID: "uuid", AppURL: mockAppURL},
			{UUID: "uuid", AppURL: mockAppURL},
		}
		mockDb.On("GetTribesByAppUrl", mockAppURL).Return(mockTribes).Once()

		// Serve request
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("app_url", mockAppURL)
		req, err := http.NewRequestWithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx), http.MethodGet, "/app_url/"+mockAppURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		handler := http.HandlerFunc(tHandler.GetTribesByAppUrl)
		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData []db.Tribe
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("Error decoding JSON response: %s", err)
		}
		assert.ElementsMatch(t, mockTribes, responseData)
	})
}

func TestDeleteTribe(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.ContextKey, "owner_pubkey")
	mockDb := mocks.NewDatabase(t)
	tHandler := NewTribeHandler(mockDb)

	t.Run("Should test that the owner of a tribe can delete a tribe", func(t *testing.T) {
		// Mock data
		mockUUID := "valid_uuid"
		mockOwnerPubKey := "owner_pubkey"

		mockVerifyTribeUUID := func(uuid string, checkTimestamp bool) (string, error) {
			return mockOwnerPubKey, nil
		}
		mockDb.On("UpdateTribe", mock.Anything, map[string]interface{}{"deleted": true}).Return(true)

		tHandler.verifyTribeUUID = mockVerifyTribeUUID

		// Create and serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.DeleteTribe)

		req, err := http.NewRequestWithContext(ctx, "DELETE", "/tribe/"+mockUUID, nil)
		if err != nil {
			t.Fatal(err)
		}
		chiCtx := chi.NewRouteContext()
		chiCtx.URLParams.Add("uuid", "mockUUID")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData bool
		errors := json.Unmarshal(rr.Body.Bytes(), &responseData)
		assert.NoError(t, errors)
		assert.True(t, responseData)
	})

	t.Run("Should test that a 401 error is returned when a tribe is attempted to be deleted by someone other than the owner", func(t *testing.T) {
		// Mock data
		ctx := context.WithValue(context.Background(), auth.ContextKey, "pubkey")
		mockUUID := "valid_uuid"
		mockOwnerPubKey := "owner_pubkey"

		mockVerifyTribeUUID := func(uuid string, checkTimestamp bool) (string, error) {
			return mockOwnerPubKey, nil
		}

		tHandler.verifyTribeUUID = mockVerifyTribeUUID

		// Create and serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.DeleteTribe)

		req, err := http.NewRequestWithContext(ctx, "DELETE", "/tribe/"+mockUUID, nil)
		if err != nil {
			t.Fatal(err)
		}
		chiCtx := chi.NewRouteContext()
		chiCtx.URLParams.Add("uuid", "mockUUID")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestGetFirstTribeByFeed(t *testing.T) {
	mockDb := mocks.NewDatabase(t)
	tHandler := NewTribeHandler(mockDb)

	t.Run("Should test that a tribe can be gotten by passing the feed URL", func(t *testing.T) {
		// Mock data
		mockFeedURL := "valid_feed_url"
		mockTribe := db.Tribe{
			UUID: "valid_uuid",
		}
		mockChannels := []db.Channel{
			{ID: 1, TribeUUID: mockTribe.UUID},
		}

		mockDb.On("GetFirstTribeByFeedURL", mockFeedURL).Return(mockTribe).Once()
		mockDb.On("GetChannelsByTribe", mockTribe.UUID).Return(mockChannels).Once()

		// Create request with valid feed URL
		req, err := http.NewRequest("GET", "/tribe_by_feed?url="+mockFeedURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.GetFirstTribeByFeed)
		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("Error decoding JSON response: %s", err)
		}
		assert.Equal(t, mockTribe.UUID, responseData["uuid"])
	})
}

func TestSetTribePreview(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.ContextKey, "owner_pubkey")
	mockDb := mocks.NewDatabase(t)
	tHandler := NewTribeHandler(mockDb)

	t.Run("Should test that the owner of a tribe can set tribe preview", func(t *testing.T) {
		// Mock data
		mockUUID := "valid_uuid"
		mockOwnerPubKey := "owner_pubkey"

		mockVerifyTribeUUID := func(uuid string, checkTimestamp bool) (string, error) {
			return mockOwnerPubKey, nil
		}
		mockDb.On("UpdateTribe", mock.Anything, map[string]interface{}{"preview": "preview"}).Return(true)

		tHandler.verifyTribeUUID = mockVerifyTribeUUID

		// Create and serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.SetTribePreview)

		req, err := http.NewRequestWithContext(ctx, "PUT", "/tribepreview/"+mockUUID+"?preview=preview", nil)
		if err != nil {
			t.Fatal(err)
		}
		chiCtx := chi.NewRouteContext()
		chiCtx.URLParams.Add("uuid", "mockUUID")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		var responseData bool
		errors := json.Unmarshal(rr.Body.Bytes(), &responseData)
		assert.NoError(t, errors)
		assert.True(t, responseData)
	})

	t.Run("Should test that a 401 error is returned when setting a tribe preview action by someone other than the owner", func(t *testing.T) {
		// Mock data
		ctx := context.WithValue(context.Background(), auth.ContextKey, "pubkey")
		mockUUID := "valid_uuid"
		mockOwnerPubKey := "owner_pubkey"

		mockVerifyTribeUUID := func(uuid string, checkTimestamp bool) (string, error) {
			return mockOwnerPubKey, nil
		}

		tHandler.verifyTribeUUID = mockVerifyTribeUUID

		// Create and serve request
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(tHandler.SetTribePreview)

		req, err := http.NewRequestWithContext(ctx, "PUT", "/tribepreview/"+mockUUID+"?preview=preview", nil)
		if err != nil {
			t.Fatal(err)
		}
		chiCtx := chi.NewRouteContext()
		chiCtx.URLParams.Add("uuid", "mockUUID")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}