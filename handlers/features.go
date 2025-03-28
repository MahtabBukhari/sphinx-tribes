package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/stakwork/sphinx-tribes/websocket"

	"github.com/stakwork/sphinx-tribes/utils"

	"os"

	"github.com/go-chi/chi"
	"github.com/rs/xid"
	"github.com/stakwork/sphinx-tribes/auth"
	"github.com/stakwork/sphinx-tribes/db"
	"github.com/stakwork/sphinx-tribes/logger"
)

type PostData struct {
	ProductBrief      string   `json:"productBrief"`
	FeatureName       string   `json:"featureName"`
	Description       string   `json:"description"`
	Examples          []string `json:"examples"`
	WebhookURL        string   `json:"webhook_url"`
	FeatureUUID       string   `json:"featureUUID"`
	Alias             string   `json:"alias"`
	SourceWebsocketId string   `json:"sourceWebsocketId"`
}

type FeatureCallRequest struct {
    WorkspaceID string `json:"workspace_id"`
    URL         string `json:"url"`
}

type FeatureBriefRequest struct {
	Output struct {
		FeatureBrief string `json:"featureBrief"`
		AudioLink    string `json:"audioLink"`
		FeatureUUID  string `json:"featureUUID"`
	} `json:"output"`
}
type AudioBriefPostData struct {
	AudioLink   string   `json:"audioLink"`
	FeatureUUID string   `json:"featureUUID"`
	Source      string   `json:"source"`
	Examples    []string `json:"examples"`
}

type featureHandler struct {
	db                    db.Database
	generateBountyHandler func(bounties []db.NewBounty) []db.BountyResponse
}

func NewFeatureHandler(database db.Database) *featureHandler {
	bHandler := NewBountyHandler(http.DefaultClient, database)
	return &featureHandler{
		db:                    database,
		generateBountyHandler: bHandler.GenerateBountyResponse,
	}
}

// CreateOrEditFeatures godoc
//
//	@Summary		Create or Edit Features
//	@Description	Create or edit features
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.WorkspaceFeatures
//	@Router			/features [post]
func (oh *featureHandler) CreateOrEditFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	features := db.WorkspaceFeatures{}
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	err := json.Unmarshal(body, &features)

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	features.CreatedBy = pubKeyFromAuth

	if features.Uuid == "" {
		features.Uuid = xid.New().String()
		features.FeatStatus = db.ActiveFeature
	} else {
		features.UpdatedBy = pubKeyFromAuth
	}

	// Validate struct data
	err = db.Validate.Struct(features)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf("Error: did not pass validation test : %s", err)
		json.NewEncoder(w).Encode(msg)
		return
	}

	// Check if workspace exists
	workpace := oh.db.GetWorkspaceByUuid(features.WorkspaceUuid)
	if workpace.Uuid != features.WorkspaceUuid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("Workspace does not exists")
		return
	}

	p, err := oh.db.CreateOrEditFeature(features)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(p)
}

// DeleteFeature godoc
//
//	@Summary		Delete Feature
//	@Description	Delete a feature by its UUID
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{string}	string	"Feature deleted successfully"
//	@Router			/features/{uuid} [delete]
func (oh *featureHandler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	uuid := chi.URLParam(r, "uuid")
	err := oh.db.DeleteFeatureByUuid(uuid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Feature deleted successfully")
}

// GetFeaturesByWorkspaceUuid godoc
//
//	@Summary		Get Features by Workspace UUID
//	@Description	Get features by workspace UUID
//	@Tags			Feature - Workspaces
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{array}	db.WorkspaceFeatures
//	@Router			/features/forworkspace/{workspace_uuid} [get]
func (oh *featureHandler) GetFeaturesByWorkspaceUuid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	uuid := chi.URLParam(r, "workspace_uuid")
	workspaceFeatures := oh.db.GetFeaturesByWorkspaceUuid(uuid, r)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(workspaceFeatures)
}

// GetWorkspaceFeaturesCount godoc
//
//	@Summary		Get Workspace Features Count
//	@Description	Get the count of features in a workspace
//	@Tags			Feature - Workspaces
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{int}	int
//	@Router			/features/workspace/count/{uuid} [get]
func (oh *featureHandler) GetWorkspaceFeaturesCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !utils.ValidateUUID(r) {
		logger.Log.Info("invalid or missing uuid")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid or missing uuid"})
		return
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		logger.Log.Info("missing or empty uuid")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing or empty uuid"})
		return
	}

	workspaceFeatures := oh.db.GetWorkspaceFeaturesCount(uuid)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(workspaceFeatures)
}

// GetFeatureByUuid godoc
//
//	@Summary		Get Feature by UUID
//	@Description	Get a feature by its UUID
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.WorkspaceFeatures
//	@Router			/features/{uuid} [get]
func (oh *featureHandler) GetFeatureByUuid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)

	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	uuid := chi.URLParam(r, "uuid")

	if uuid == "" {
		logger.Log.Info("missing uuid parameter")
		http.Error(w, "uuid parameter is required", http.StatusBadRequest)
		return
	}

	workspaceFeature := oh.db.GetFeatureByUuid(uuid)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(workspaceFeature)
}

// UpdateFeatureStatus godoc
//
//	@Summary		Update Feature Brief
//	@Description	Update the brief of a feature
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.WorkspaceFeatures
//	@Router			/features/brief [post]
func (oh *featureHandler) UpdateFeatureBrief(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req FeatureBriefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid request payload")
		return
	}

	featureUUID := req.Output.FeatureUUID
	newFeatureBrief := req.Output.FeatureBrief

	if featureUUID == "" || newFeatureBrief == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Missing required fields")
		return
	}

	prevFeatureBrief := oh.db.GetFeatureByUuid(featureUUID)

	if prevFeatureBrief.Uuid == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Feature not found")
		return
	}

	var updatedFeatureBrief string
	if prevFeatureBrief.Brief == "" {
		updatedFeatureBrief = newFeatureBrief
	} else {

		updatedFeatureBrief = prevFeatureBrief.Brief + "\n\n* Generated Feature Brief *\n\n" + newFeatureBrief
	}

	featureToUpdate := db.WorkspaceFeatures{
		Uuid:                   featureUUID,
		WorkspaceUuid:          prevFeatureBrief.WorkspaceUuid,
		Name:                   prevFeatureBrief.Name,
		Brief:                  updatedFeatureBrief,
		Requirements:           prevFeatureBrief.Requirements,
		Architecture:           prevFeatureBrief.Architecture,
		Url:                    prevFeatureBrief.Url,
		Priority:               prevFeatureBrief.Priority,
		BountiesCountCompleted: prevFeatureBrief.BountiesCountCompleted,
		BountiesCountAssigned:  prevFeatureBrief.BountiesCountAssigned,
		BountiesCountOpen:      prevFeatureBrief.BountiesCountOpen,
	}

	p, err := oh.db.CreateOrEditFeature(featureToUpdate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(p)
}

// CreateOrEditFeaturePhase godoc
//
//	@Summary		Create or Edit Feature Phase
//	@Description	Create or edit a phase of a feature
//	@Tags			Feature - Phases
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		201	{object}	db.FeaturePhase
//	@Router			/features/phase [post]
func (oh *featureHandler) CreateOrEditFeaturePhase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newPhase := db.FeaturePhase{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newPhase)
	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "Error decoding request body: %v", err)
		return
	}

	if newPhase.Uuid == "" {
		newPhase.Uuid = xid.New().String()
	}

	existingPhase, _ := oh.db.GetFeaturePhaseByUuid(newPhase.FeatureUuid, newPhase.Uuid)

	if existingPhase.CreatedBy == "" {
		newPhase.CreatedBy = pubKeyFromAuth
	}

	newPhase.UpdatedBy = pubKeyFromAuth

	// Check if feature exists
	feature := oh.db.GetFeatureByUuid(newPhase.FeatureUuid)
	if feature.Uuid != newPhase.FeatureUuid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("Feature does not exists")
		return
	}

	phase, err := oh.db.CreateOrEditFeaturePhase(newPhase)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error creating feature phase: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(phase)
}

// GetFeaturePhases godoc
//
//	@Summary		Get Feature Phases
//	@Description	Get phases of a feature by its UUID
//	@Tags			Feature - Phases
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{array}	db.FeaturePhase
//	@Router			/features/{feature_uuid}/phase [get]
func (oh *featureHandler) GetFeaturePhases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUuid := chi.URLParam(r, "feature_uuid")
	phases := oh.db.GetPhasesByFeatureUuid(featureUuid)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(phases)
}

// GetFeaturePhaseByUUID godoc
//
//	@Summary		Get Feature Phase by UUID
//	@Description	Get a phase of a feature by its UUID
//	@Tags			Feature - Phases
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.FeaturePhase
//	@Router			/features/{feature_uuid}/phase/{phase_uuid} [get]
func (oh *featureHandler) GetFeaturePhaseByUUID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	person := oh.db.GetPersonByPubkey(pubKeyFromAuth)
	if person.OwnerPubKey != pubKeyFromAuth {
		logger.Log.Info("Invalid pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUuid := chi.URLParam(r, "feature_uuid")
	phaseUuid := chi.URLParam(r, "phase_uuid")

	phase, err := oh.db.GetFeaturePhaseByUuid(featureUuid, phaseUuid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(phase)
}

// DeleteFeaturePhase godoc
//
//	@Summary		Delete Feature Phase
//	@Description	Delete a phase of a feature by its UUID
//	@Tags			Feature - Phases
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{string}	string	"Phase deleted successfully"
//	@Router			/features/{feature_uuid}/phase/{phase_uuid} [delete]
func (oh *featureHandler) DeleteFeaturePhase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUuid := chi.URLParam(r, "feature_uuid")
	phaseUuid := chi.URLParam(r, "phase_uuid")

	if !isValidUUID(featureUuid) || !isValidUUID(phaseUuid) {
		logger.Log.Info("Malformed UUIDs")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Malformed UUIDs"})
		return
	}

	err := oh.db.DeleteFeaturePhase(featureUuid, phaseUuid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Phase deleted successfully"})
}

// CreateOrEditStory godoc
//
//	@Summary		Create or Edit Story
//	@Description	Create or edit a story of a feature
//	@Tags			Feature - Stories
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		201	{object}	db.FeatureStory
//	@Router			/features/story [post]
func (oh *featureHandler) CreateOrEditStory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newStory := db.FeatureStory{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newStory)
	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "Error decoding request body: %v", err)
		return
	}

	if newStory.Uuid == "" {
		newStory.Uuid = xid.New().String()
	}

	existingStory, _ := oh.db.GetFeatureStoryByUuid(newStory.FeatureUuid, newStory.Uuid)

	if existingStory.CreatedBy == "" {
		newStory.CreatedBy = pubKeyFromAuth
	}

	newStory.UpdatedBy = pubKeyFromAuth

	story, err := oh.db.CreateOrEditFeatureStory(newStory)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error creating feature story: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(story)
}

// GetStoriesByFeatureUuid godoc
//
//	@Summary		Get Stories by Feature UUID
//	@Description	Get stories of a feature by its UUID
//	@Tags			Feature - Stories
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{array}	db.FeatureStory
//	@Router			/features/{feature_uuid}/story [get]
func (oh *featureHandler) GetStoriesByFeatureUuid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUuid := chi.URLParam(r, "feature_uuid")
	if featureUuid == "" {
		logger.Log.Info("empty feature uuid")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	stories, err := oh.db.GetFeatureStoriesByFeatureUuid(featureUuid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stories)
}

// GetStoryByUuid godoc
//
//	@Summary		Get Story by UUID
//	@Description	Get a story of a feature by its UUID
//	@Tags			Feature - Stories
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.FeatureStory
//	@Router			/features/{feature_uuid}/story/{story_uuid} [get]
func (oh *featureHandler) GetStoryByUuid(w http.ResponseWriter, r *http.Request) {
	featureUuid := chi.URLParam(r, "feature_uuid")
	storyUuid := chi.URLParam(r, "story_uuid")

	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	story, err := oh.db.GetFeatureStoryByUuid(featureUuid, storyUuid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(story)
}

// DeleteStory godoc
//
//	@Summary		Delete Story
//	@Description	Delete a story of a feature by its UUID
//	@Tags			Feature - Stories
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{string}	string	"Story deleted successfully"
//	@Router			/features/{feature_uuid}/story/{story_uuid} [delete]
func (oh *featureHandler) DeleteStory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUuid := chi.URLParam(r, "feature_uuid")
	storyUuid := chi.URLParam(r, "story_uuid")

	err := oh.db.DeleteFeatureStoryByUuid(featureUuid, storyUuid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Story deleted successfully"})
}

// GetBountiesByFeatureAndPhaseUuid godoc
//
//	@Summary		Get Bounties by Feature and Phase UUID
//	@Description	Get bounties of a feature by its UUID and phase UUID
//	@Tags			Feature - Phases
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{array}	db.BountyResponse
//	@Router			/features/{feature_uuid}/phase/{phase_uuid}/bounty [get]
func (oh *featureHandler) GetBountiesByFeatureAndPhaseUuid(w http.ResponseWriter, r *http.Request) {
	featureUuid := chi.URLParam(r, "feature_uuid")
	phaseUuid := chi.URLParam(r, "phase_uuid")

	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	bounties, err := oh.db.GetBountiesByFeatureAndPhaseUuid(featureUuid, phaseUuid, r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bountyResponses := oh.generateBountyHandler(bounties)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bountyResponses)
}

// GetBountiesCountByFeatureAndPhaseUuid godoc
//
//	@Summary		Get Bounties Count by Feature and Phase UUID
//	@Description	Get the count of bounties of a feature by its UUID and phase UUID
//	@Tags			Feature - Phases
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{int}	int
//	@Router			/features/{feature_uuid}/phase/{phase_uuid}/bounty/count [get]
func (oh *featureHandler) GetBountiesCountByFeatureAndPhaseUuid(w http.ResponseWriter, r *http.Request) {
	featureUuid := chi.URLParam(r, "feature_uuid")
	phaseUuid := chi.URLParam(r, "phase_uuid")

	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	bountiesCount := oh.db.GetBountiesCountByFeatureAndPhaseUuid(featureUuid, phaseUuid, r)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bountiesCount)
}

// GetFeatureStories godoc
//
//	@Summary		Get Feature Stories
//	@Description	Get stories for a feature
//	@Tags			Feature - Stories
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	db.FeatureStoriesReponse
//	@Router			/features/stories [post]
func (oh *featureHandler) GetFeatureStories(w http.ResponseWriter, r *http.Request) {
	featureStories := db.FeatureStoriesReponse{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&featureStories)

	featureUuid := featureStories.Output.FeatureUuid

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		log.Printf("Error decoding request body: %v", err)
		return
	}

	log.Println("Webhook Feature Uuid", featureUuid)

	log.Println("Webhook Feature Stories === ", featureStories.Output.Stories)

	// check if feature story exists
	feature := oh.db.GetFeatureByUuid(featureUuid)

	if feature.ID == 0 {
		msg := "Feature ID does not exists"
		log.Println(msg, featureUuid)
		w.WriteHeader(http.StatusNotAcceptable)
		json.NewEncoder(w).Encode(msg)
		return
	}

	for _, story := range featureStories.Output.Stories {

		now := time.Now()

		// Add story to database
		featureStory := db.FeatureStory{
			Uuid:        xid.New().String(),
			Description: story.UserStory,
			FeatureUuid: featureUuid,
			Created:     &now,
			Updated:     &now,
		}

		oh.db.CreateOrEditFeatureStory(featureStory)
		log.Println("Created user story for : ", featureStory.FeatureUuid)
	}

	ticketMsg := websocket.TicketMessage{
		BroadcastType:   "direct",
		SourceSessionID: featureStories.Output.SourceWebsocketId,
		Message:         fmt.Sprintf("Successfully created new user stories"),
		Action:          "process",
	}

	if err := websocket.WebsocketPool.SendTicketMessage(ticketMsg); err != nil {
		log.Printf("Failed to send websocket message: %v", err)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Failed to send websocket message")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("User stories added successfully")
}

// SendStories godoc
//
//	@Summary		Send Stories
//	@Description	Send stories of a feature
//	@Tags			Feature - Stories
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{string}	string	"Successfully sent"
//	@Router			/features/stories/send [post]
func (oh *featureHandler) StoriesSend(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user := oh.db.GetPersonByPubkey(pubKeyFromAuth)

	if user.OwnerPubKey != pubKeyFromAuth {
		logger.Log.Info("Person not exists")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to read requests body", http.StatusBadRequest)
		return
	}

	var postData PostData
	err = json.Unmarshal(body, &postData)
	if err != nil {
		logger.Log.Error("[StoriesSend] JSON Unmarshal error: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusNotAcceptable)
		return
	}

	apiKey := os.Getenv("SWWFKEY")
	if apiKey == "" {
		panic("API key not set in environment")
	}

	postData.Alias = user.OwnerAlias

	stakworkPayload := map[string]interface{}{
		"name":        "string",
		"workflow_id": 35080,
		"workflow_params": map[string]interface{}{
			"set_var": map[string]interface{}{
				"attributes": map[string]interface{}{
					"vars": postData,
				},
			},
		},
	}

	stakworkPayloadJSON, err := json.Marshal(stakworkPayload)
	if err != nil {
		panic("Failed to encode payload")
	}

	req, err := http.NewRequest("POST", "https://api.stakwork.com/api/v1/projects", bytes.NewBuffer(stakworkPayloadJSON))
	if err != nil {
		panic("Failed to create request to Stakwork API")
	}
	req.Header.Set("Authorization", "Token token="+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic("Failed to send request to Stakwork API")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Failed to read response from Stakwork API")
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// BriefSend godoc
//
//	@Summary		Send Feature Brief
//	@Description	Send the brief of a feature
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{string}	string	"Successfully sent"
//	@Router			/features/brief/send [post]
func (oh *featureHandler) BriefSend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user := oh.db.GetPersonByPubkey(pubKeyFromAuth)

	if user.OwnerPubKey != pubKeyFromAuth {
		logger.Log.Info("Person not exists")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to read requests body", http.StatusBadRequest)
		return
	}

	var postData AudioBriefPostData
	err = json.Unmarshal(body, &postData)
	if err != nil {
		logger.Log.Error("[BriefSend] JSON Unmarshal error: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusNotAcceptable)
		return
	}

	host := os.Getenv("HOST")
	if host == "" {
		logger.Log.Error("[BriefSend] HOST environment variable not set")
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	completePostData := struct {
		AudioBriefPostData
		WebhookURL string `json:"webhook_url"`
		Alias      string `json:"alias"`
	}{
		AudioBriefPostData: postData,
		WebhookURL:         fmt.Sprintf("%s/feature/brief", host),
		Alias:              user.OwnerAlias,
	}

	apiKey := os.Getenv("SWWFKEY")
	if apiKey == "" {
		logger.Log.Error("[BriefSend] API key not set in environment")
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	stakworkPayload := map[string]interface{}{
		"name":        "string",
		"workflow_id": 36928,
		"workflow_params": map[string]interface{}{
			"set_var": map[string]interface{}{
				"attributes": map[string]interface{}{
					"vars": completePostData,
				},
			},
		},
	}

	stakworkPayloadJSON, err := json.Marshal(stakworkPayload)
	if err != nil {
		panic("Failed to encode payload")
		return
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.stakwork.com/api/v1/projects", bytes.NewBuffer(stakworkPayloadJSON))
	if err != nil {
		panic("Failed to create request to Stakwork API")
		return
	}
	req.Header.Set("Authorization", "Token token="+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic("Failed to send request to Stakwork API")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Failed to read response from Stakwork API")
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// UpdateFeatureStatus godoc
//
//	@Summary		Update Feature Status
//	@Description	Update the status of a feature
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.WorkspaceFeatures
//	@Router			/features/{uuid}/status [put]
func (oh *featureHandler) UpdateFeatureStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	person := oh.db.GetPersonByPubkey(pubKeyFromAuth)
	if person.OwnerPubKey == "" {
		logger.Log.Info("invalid pubkey")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unauthorized: invalid pubkey",
		})
		return
	}

	uuid := chi.URLParam(r, "uuid")

	if uuid == "" {
		logger.Log.Info("uuid parameter is missing")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing uuid parameter",
		})
		return
	}

	if r.Body == nil {
		logger.Log.Info("request body is nil")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Request body is required",
		})
		return
	}

	var req struct {
		Status db.FeatureStatus `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("invalid request body", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, valid := map[db.FeatureStatus]bool{
		db.ActiveFeature:    true,
		db.ArchivedFeature:  true,
		db.CompletedFeature: true,
		db.BacklogFeature:   true,
	}[req.Status]; !valid {
		logger.Log.Info("invalid feature status")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid feature status. Allowed values are: active, archived, completed, backlog",
		})
		return
	}

	updatedFeature, err := oh.db.UpdateFeatureStatus(uuid, req.Status)
	if err != nil {
		logger.Log.Error("failed to update feature status", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedFeature)
}

// GetQuickBounties godoc
//
//	@Summary		Get Quick Bounties
//	@Description	Get quick bounties of a feature by its UUID
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.QuickBountiesResponse
//	@Router			/features/{feature_uuid}/quick-bounties [get]
func (oh *featureHandler) GetQuickBounties(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUUID := chi.URLParam(r, "feature_uuid")
	if featureUUID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "feature_uuid is required"})
		return
	}

	feature := oh.db.GetFeatureByUuid(featureUUID)
	if feature.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "feature not found"})
		return
	}

	bounties, err := oh.db.GetBountiesByFeatureUuid(featureUUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := db.QuickBountiesResponse{
		FeatureID: featureUUID,
		Phases:    make(map[string][]db.QuickBountyItem),
		Unphased:  make([]db.QuickBountyItem, 0),
	}

	for _, bounty := range bounties {
		var assignedAlias *string
		if bounty.Assignee != "" {
			assignee := oh.db.GetPersonByPubkey(bounty.Assignee)
			if assignee.OwnerAlias != "" {
				assignedAlias = &assignee.OwnerAlias
			}
		}

		status := calculateBountyStatus(bounty)

		var phaseID *string
		if bounty.PhaseUuid != "" {
			phaseUUID := bounty.PhaseUuid
			phaseID = &phaseUUID
		}

		item := db.QuickBountyItem{
			BountyID:      bounty.ID,
			BountyTitle:   bounty.Title,
			Status:        status,
			AssignedAlias: assignedAlias,
			PhaseID:       phaseID,
		}

		if status == db.StatusTodo {
			item.AssignedAlias = nil
		}

		if bounty.PhaseUuid != "" {
			response.Phases[bounty.PhaseUuid] = append(response.Phases[bounty.PhaseUuid], item)
		} else {
			response.Unphased = append(response.Unphased, item)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetQuickTickets godoc
//
//	@Summary		Get Quick Tickets
//	@Description	Get quick tickets of a feature by its UUID
//	@Tags			Features
//	@Accept			json
//	@Produce		json
//	@Security		PubKeyContextAuth
//	@Success		200	{object}	db.QuickTicketsResponse
//	@Router			/features/{feature_uuid}/quick-tickets [get]
func (oh *featureHandler) GetQuickTickets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	featureUUID := chi.URLParam(r, "feature_uuid")
	if featureUUID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "feature_uuid is required"})
		return
	}

	feature := oh.db.GetFeatureByUuid(featureUUID)
	if feature.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "feature not found"})
		return
	}

	tickets, err := oh.db.GetTicketsByFeatureUUID(featureUUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := db.QuickTicketsResponse{
		FeatureID: featureUUID,
		Phases:    make(map[string][]db.QuickTicketItem),
		Unphased:  make([]db.QuickTicketItem, 0),
	}

	latestTickets := make(map[string]db.QuickTicketItem)

	for _, ticket := range tickets {
		var phaseID *string
		if ticket.PhaseUUID != "" {
			phaseUUID := ticket.PhaseUUID
			phaseID = &phaseUUID
		}

		item := db.QuickTicketItem{
			TicketUUID:    ticket.UUID,
			TicketTitle:   ticket.Name,
			Status:        db.StatusDraft,
			AssignedAlias: nil,
			PhaseID:       phaseID,
		}

		if existingItem, exists := latestTickets[ticket.TicketGroup.String()]; !exists || ticket.UUID.String() > existingItem.TicketUUID.String() {
			latestTickets[ticket.TicketGroup.String()] = item
		}
	}

	for _, item := range latestTickets {
		if item.PhaseID != nil {
			response.Phases[*item.PhaseID] = append(response.Phases[*item.PhaseID], item)
		} else {
			response.Unphased = append(response.Unphased, item)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (oh *featureHandler) CreateOrUpdateFeatureCall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req FeatureCallRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("invalid request body", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	workspace := oh.db.GetWorkspaceByUuid(req.WorkspaceID)
	if workspace.Uuid == "" {
		logger.Log.Info("workspace not found")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Workspace not found"})
		return
	}

	featureCall, err := oh.db.CreateOrUpdateFeatureCall(req.WorkspaceID, req.URL)
	if err != nil {
		logger.Log.Error("failed to create/update feature call", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(featureCall)
}

func (oh *featureHandler) GetFeatureCall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	workspaceID := chi.URLParam(r, "workspace_uuid")
	if workspaceID == "" {
		logger.Log.Info("missing workspace_uuid parameter")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workspace_uuid parameter is required"})
		return
	}

	featureCall, err := oh.db.GetFeatureCallByWorkspaceID(workspaceID)
	if err != nil {
		logger.Log.Error("failed to get feature call", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(featureCall)
}

func (oh *featureHandler) DeleteFeatureCall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	if pubKeyFromAuth == "" {
		logger.Log.Info("no pubkey from auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	workspaceID := chi.URLParam(r, "workspace_uuid")
	if workspaceID == "" {
		logger.Log.Info("missing workspace_uuid parameter")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workspace_uuid parameter is required"})
		return
	}

	err := oh.db.DeleteFeatureCall(workspaceID)
	if err != nil {
		logger.Log.Error("failed to delete feature call", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Feature call deleted successfully"})
}
